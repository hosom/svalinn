# svalinn
Svalinn is a [Windows Password Filter]("https://msdn.microsoft.com/en-us/library/windows/desktop/ms721882(v=vs.85).aspx"). 

## Installation

Svalinn can be installed by following the instructions below.

1. Add `svalinn.dll` to your `%SYSTEMROOT%\System32` directory.
2. Modify the registry key located at `HKEY_LOCAL_MACHINE\SYSTEM\CurrentControlSet\Control\Lsa` and add `svalinn` as an entry. 
    **Do not remove any of the current entries in this registry key.**
3. Reboot

### Install Verification

Once rebooted, you can verify that svalinn is loaded by looking at the output of **msinfo32.exe**. You should see **svalin.dll** as a Loaded module within **Software Environment | Loaded Modules**. 

Additionally, you can verify the status of the password filter by perrforming a password reset and svalinn will log whether the reset fails or is successful within the Application event log.

If the log message does not appear, something has gone wrong in the process of loading svalinn. To troubleshoot, look at the System event log for entries stating that a password filter failed to load. 

## Configuring

Svalinn is configured via the registry. The appropriate configuration keys should be stored at `HKEY_LOCAL_MACHINE\SOFTWARE\passfilt`.

All configuration values must be specified and within valid ranges. **Failure to configure the registry keys properly will result in svalinn defaulting to allowing password changes based on the Domain policy alone.**

The required values are:

**Server**: The server to send password requests to.

**Port**: The TCP port to connect to.

**Enable TLS**: Enable TLS on connections to the password filter server. **STRONGLY RECOMMENDED**.

**Disable TLS Validation**: Disable validation of TLS certificates. **STRONGLY DISCOURAGED**.

Please note that in production environments it is **strongly** recommended that users enable TLS and do not disable TLS validation. Misconfiguration of these values can drastically increase the risk of a man-in-the-middle attack intercepting passwords.

## Removal

1. Remove the **svalinn** entry from `HKEY_LOCAL_MACHINE\SYSTEM\CurrentControlSet\Control\Lsa`. 
2. Reboot
3. [Optional] Remove `svalinn.dll` from `%SYSTEMROOT\System32`.
    * Note: You cannot remove the dll until after a reboot. It must first be unloaded by the lsass process.


#### Noteworthy Work

This is not the first open source password filter. I have drawn inspiration from the following sources:

* [OpenPasswordFilter](https://github.com/jephthai/OpenPasswordFilter)
* [CredDefense](https://github.com/CredDefense/CredDefense)