// dllmain.cpp : Defines the entry point for the DLL application.
#include "stdafx.h"
#include <NTSecAPI.h> // PUNICODE_STRING
#include <string> // string operations
#include <winhttp.h> // http actions
#include <sstream> // logging
#include <iostream> // logging

#pragma comment(lib, "winhttp.lib") // required for winhttp

// SUBKEY is where passfilt configuration is stored
const std::wstring SUBKEY = L"SOFTWARE\\passfilt";
// EVTLOG_NAME defines the name for winevt logging
const std::wstring EVTLOG_NAME = L"svalinn";

/* standard dll boilerplate */
BOOL APIENTRY DllMain( HMODULE hModule,
                       DWORD  ul_reason_for_call,
                       LPVOID lpReserved
                     )
{
    switch (ul_reason_for_call)
    {
    case DLL_PROCESS_ATTACH:
    case DLL_THREAD_ATTACH:
    case DLL_THREAD_DETACH:
    case DLL_PROCESS_DETACH:
        break;
    }
    return TRUE;
}/* standard dll boilerplate */

// config_t represents the configuration read from the registry
struct config_t {
	std::wstring srv;
	DWORD port;
	DWORD enableTLS;
	DWORD disableTLSValidation;
};

// logMsg will log a message to the winevt log 
// EVENTLOG_ERROR_TYPE or EVENTLOG_SUCCESS are the most likely
// desired msgTypes
void logMsg(DWORD msgType, LPCWSTR msg) {
	// I'm really not sure what the best way to deal with a failure
	// to open the event log is... I won't crash, because this 
	// doesn't do anything on a failure. Unfortunately, it also
	// means that you may have no way to know that you are not 
	// logging errors.
	HANDLE evt_log = RegisterEventSource(NULL, EVTLOG_NAME.c_str());
	if (evt_log) {
		ReportEvent(evt_log, msgType, 0, 0, NULL, 1, 0, &msg, NULL);
		DeregisterEventSource(evt_log);
	}
}

// readConfig parses the configuration from the registry and passes
// it back in the provided config_t struct
 LONG readConfig(config_t *conf) {
	 LONG rCode; // response code from registry actions
	 DWORD dataSize{}; // buffer size for registry actions
	 std::wstring msg; // log message to send to winevt
	 
	 // Get the server address...
	 std::wstring value = L"server";
	 rCode = RegGetValue(HKEY_LOCAL_MACHINE, 
		 SUBKEY.c_str(), 
		 value.c_str(), 
		 RRF_RT_REG_SZ, 
		 nullptr, 
		 nullptr, 
		 &dataSize);

	 if (rCode != ERROR_SUCCESS) {
		 msg = L"Failed to obtain size of server registry value.";
		 goto CLEANUP;
	 }

	 // change the size of srv to receive the value of the registry key
	 conf->srv.resize(dataSize / sizeof(wchar_t));
	 rCode = RegGetValue(
		 HKEY_LOCAL_MACHINE,
		 SUBKEY.c_str(),
		 value.c_str(),
		 RRF_RT_REG_SZ,
		 nullptr,
		 &conf->srv[0],
		 &dataSize
	 );

	 if (rCode != ERROR_SUCCESS) {
		 msg = L"Failed to retrieve value for server from registry.";
		 goto CLEANUP;
	 }

	 // retrieve the port to access
	 value = L"port";
	 dataSize = sizeof(conf->port);
	 rCode = RegGetValue(
		 HKEY_LOCAL_MACHINE,
		 SUBKEY.c_str(),
		 value.c_str(),
		 RRF_RT_REG_DWORD,
		 nullptr,
		 &conf->port,
		 &dataSize
	 );

	 if (rCode != ERROR_SUCCESS) {
		 msg = L"Failed to read value for port from registry.";
		 goto CLEANUP;
	 }

	 // retrieve whether or not to enable TLS
	 value = L"enable tls";
	 dataSize = sizeof(conf->enableTLS);
	 rCode = RegGetValue(
		 HKEY_LOCAL_MACHINE,
		 SUBKEY.c_str(),
		 value.c_str(),
		 RRF_RT_REG_DWORD,
		 nullptr,
		 &conf->enableTLS,
		 &dataSize
	 );

	 if (rCode != ERROR_SUCCESS) {
		 msg = L"Failed to read TLS configuration from registry.";
		 goto CLEANUP;
	 }

	 // retrieve whether or not to enable TLS
	 value = L"disable tls validation";
	 dataSize = sizeof(conf->disableTLSValidation);
	 rCode = RegGetValue(
		 HKEY_LOCAL_MACHINE,
		 SUBKEY.c_str(),
		 value.c_str(),
		 RRF_RT_REG_DWORD,
		 nullptr,
		 &conf->disableTLSValidation,
		 &dataSize
	 );

	 if (rCode != ERROR_SUCCESS) {
		 msg = L"Failed to retrieve value for Disable TLS Validation from registry.";
		 goto CLEANUP;
	 }

 CLEANUP:
	 if (rCode != ERROR_SUCCESS) logMsg(EVENTLOG_ERROR_TYPE, msg.c_str());
	 return rCode;
}

//
// InitializeChangeNotify is used to determine if the password filter is ready
// for use. In this case, we simply return TRUE always.
//
extern "C" __declspec(dllexport) BOOLEAN __stdcall InitializeChangeNotify(void) {
	return TRUE;
}

// 
// PasswordChangeNotify notifies that a password has been changed.
// 
extern "C" __declspec(dllexport) int __stdcall
PasswordChangeNotify(PUNICODE_STRING *UserName,
	ULONG RelativeId,
	PUNICODE_STRING *NewPassword) {
	return 0;
}

//
// PasswordFilter is called during the password change process. It returns TRUE to
// permit a password change and FALSE to reject one.
//
extern "C" __declspec(dllexport) BOOLEAN __stdcall PasswordFilter(PUNICODE_STRING AccountName,
	PUNICODE_STRING FullName,
	PUNICODE_STRING Password,
	BOOLEAN SetOperation) {

	config_t conf; // parsed configuration
	std::wostringstream msg; // log message to send
	BOOL passOK = TRUE; // should the password be set?
	BOOL success = FALSE; // logging a failure or a success?
	HINTERNET hSession = NULL, // winhttp handlers
		hConnect = NULL,
		hRequest = NULL;
	DWORD dwFlags = 0; // flags for winhttp function calls
	BOOL bResults = FALSE; // winhttp results
	DWORD dwStatusCode = 0; // winhttp response code 
	DWORD dwStatusCodeSize = sizeof(dwStatusCode);
	char data = (char)"svalinndll password check";

	// convert the username and password to types usable by WinHTTP
	std::wstring user = std::wstring(AccountName->Buffer, 
		AccountName->Length / sizeof(WCHAR));
	std::wstring pass = std::wstring(Password->Buffer, 
		Password->Length / sizeof(WCHAR));

	LONG rCode = readConfig(&conf);
	if (rCode != ERROR_SUCCESS) {
		msg << L"Failed to parse configuration from the registry. Returning default TRUE." << std::endl;
		goto CLEANUP;
	}

	hSession = WinHttpOpen(L"svalinndll", 
		WINHTTP_ACCESS_TYPE_NO_PROXY, 
		WINHTTP_NO_PROXY_NAME, 
		WINHTTP_NO_PROXY_BYPASS, 0);
	if (!hSession) {
		msg << L"Failed to create hSession handle with error code: " << GetLastError() << std::endl;
		goto CLEANUP;
	}


	hConnect = WinHttpConnect(hSession, (LPCWSTR)conf.srv.c_str(), conf.port, 0);
	if (!hConnect) {
		msg << L"Failed to create hConnect handle with error code: " << GetLastError() << std::endl;
		goto CLEANUP;
	}

	// enable TLS when requested
	if (conf.enableTLS == 1) {
		dwFlags = WINHTTP_FLAG_SECURE;
	}

	hRequest = WinHttpOpenRequest(hConnect, L"POST", L"/", NULL, 
		WINHTTP_NO_REFERER, WINHTTP_DEFAULT_ACCEPT_TYPES, dwFlags);
	if (!hRequest) {
		msg << L"Failed to create hRequest handle with error code: " << GetLastError() << std::endl;
		goto CLEANUP;
	}

	// specify the credential to test
	bResults = WinHttpSetCredentials(hRequest,
		WINHTTP_AUTH_TARGET_SERVER,
		WINHTTP_AUTH_SCHEME_BASIC,
		user.c_str(),
		pass.c_str(),
		NULL);
	if (!bResults) {
		msg << L"Failed to assign credentials to request with error code: " << GetLastError() << std::endl;
		goto CLEANUP;
	}

	if (conf.enableTLS == 1 & conf.disableTLSValidation == 1) {
		// use flags in winhttp to disable certificate validation
		dwFlags = SECURITY_FLAG_IGNORE_UNKNOWN_CA |
			SECURITY_FLAG_IGNORE_CERT_WRONG_USAGE |
			SECURITY_FLAG_IGNORE_CERT_CN_INVALID |
			SECURITY_FLAG_IGNORE_CERT_DATE_INVALID;

		bResults = WinHttpSetOption(
			hRequest,
			WINHTTP_OPTION_SECURITY_FLAGS,
			&dwFlags,
			sizeof(dwFlags));
		if (!bResults) {
			msg << L"Failed to disable TLS validation with error code: " << GetLastError() << std::endl;
			goto CLEANUP;
		}
	}
	
	// finally, send the request to the API
	bResults = WinHttpSendRequest(hRequest,
		WINHTTP_NO_ADDITIONAL_HEADERS, 0, &data, 
		strlen(&data), strlen(&data), NULL);
	if (!bResults) {
			msg << L"Failed to query server for password check with error code: " << GetLastError() << std::endl;
			goto CLEANUP;
	}

	// finalize the request
	bResults = WinHttpReceiveResponse(hRequest, NULL);	
	if (!bResults) {
		msg << L"Failed to finalize response with error code: " << GetLastError() << std::endl;
		goto CLEANUP;
	}

	// parse server response
	bResults = WinHttpQueryHeaders(hRequest,
		WINHTTP_QUERY_STATUS_CODE | WINHTTP_QUERY_FLAG_NUMBER,
		WINHTTP_HEADER_NAME_BY_INDEX,
		&dwStatusCode, &dwStatusCodeSize, WINHTTP_NO_HEADER_INDEX);
	if (!bResults) {
		msg << L"Failed to parse return from server with error code: " << GetLastError() << std::endl;
		goto CLEANUP;
	}

	switch (dwStatusCode) {
		case HTTP_STATUS_OK:
			msg << L"The password has been accepted for " << user << std::endl;
			passOK = TRUE;
			goto CLEANUP;
		case HTTP_STATUS_FORBIDDEN:
			msg << L"The password server rejected the password for " << user << std::endl;
			passOK = FALSE;
			goto CLEANUP;
		default:
			msg << L"Something is very wrong. Received an unexpected return from dwStatusCode" << std::endl;
			goto CLEANUP;
	}

CLEANUP:
	// Close any open winhttp handles.
	if (hRequest) WinHttpCloseHandle(hRequest);
	if (hConnect) WinHttpCloseHandle(hConnect);
	if (hSession) WinHttpCloseHandle(hSession);

	// zero out the memory used to store the copy of the password
	SecureZeroMemory(&pass, sizeof(pass));
	
	if (success) logMsg(EVENTLOG_SUCCESS, msg.str().c_str());
	else logMsg(EVENTLOG_ERROR_TYPE, msg.str().c_str());

	return passOK;
}