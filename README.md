# svalinn
Windows Password Filter

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