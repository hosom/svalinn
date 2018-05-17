Function Passfilt {
<#
.SYNOPSIS
This Powershell script will install, uninstall, configure or check the status of Svalinn password filter. Some may find the high number of "write-host"s pointless or annoying,
this was done to make things easily understandable during the process, and ensures that 90% of open Github issues aren't "install broke, no idea what happened"

.DESCRIPTION
This is pretty straight forward, and created in a very verbose fashion (mostly) so you can tweak as necessary for additional functionality, such as email status reports, etc.
You need to run this script as a local administrator for all param switches except "-Status".

.PARAMETER -Install
This will step you through the installation process. It is required that after you complete this and reboot, you then run this script again with the "-Configure" option.

.PARAMETER -Remove
This will remove svalinn reg keys and unload the .dll module from your system.

.PARAMETER -Configure
This is required to run after you have completed the install step, or simply want to reconfigure your current svalinn instance.

.PARAMETER -Status
Utilize this option at any time to check the existence of all required configuration reg keys, their values, and .dll module load status.

.EXAMPLE
Open a powershell console as a local administrator execute script as follows (you likely will need to dot source this as seen below)

Example 1 without execution policy workarounds:
powershell -command "& { . <path>\passfilt-setup.ps1; Passfilt -Install }"

Example 2 without execution policy workarounds:
. <path>\passfilt-setup.ps1; Passfilt -Configure

Example 3 WITH execution policy workarounds:
powershell -command "& { . <path>\passfilt-setup.ps1 -ep bypass; Passfilt -Remove }"

Example 4 WITH execution policy workarounds: 
. <path>\passfilt-setup.ps1 -ep bypass; Passfilt -Status

.NOTES
#######################################################################
# Author: Tyler Currence - https://github.com/tcurrence852            #
# Project: Svalinn - Stephen Hosom - https://github.com/hosom/svalinn #
# Module Dependencies: None                                           #
# Permission level: Local Admin                                       #
# Powershell v5 or greater                                            #
#######################################################################
#>
    Param(
        [Parameter()]
        [switch]$Install,
        [Parameter()]
        [switch]$Remove,
        [Parameter()]
        [switch]$Configure,
        [Parameter()]
        [switch]$Status
    )
    If ($Install.IsPresent)
    {
        Write-Host 'Passfilt install is starting...' -ForegroundColor Cyan

        $YesOrNo = Read-Host -Prompt 'You are about to install Svalinn password filter, please confirm (Y/N)'

        If ('y', 'Y' -contains $YesOrNo)
        {
            $DLLPath = Read-Host -Prompt 'Enter directory path containing compiled svalinn.dll (ex: c:\stuff\things\)'
            $DLLName = Read-Host -Prompt 'Enter compiled svalinn DLL name without extension (ex: svalinn64)'

            Move-Item -Path ($DLLPath + $DLLName + '.dll') -Destination ($env:SystemRoot + '\System32\') -Force -ErrorAction Stop
            Sleep -Seconds 5

            $LSAReg = ((Get-ItemPropertyValue -Path 'HKLM:\SYSTEM\CurrentControlSet\Control\Lsa\' -Name 'Notification Packages') | Out-String)

            $LSARegNew = ($LSAReg + $DLLName)

            Set-ItemProperty -Path "HKLM:\SYSTEM\CurrentControlSet\Control\Lsa\" -Name "Notification Packages" -Value $LSARegNew -Type MultiString -Force -ErrorAction Stop

            Write-Host 'It is highly recommended to run -Configure at this point to finish install, followed by a reboot!' -ForegroundColor Cyan
        }
        else{
            Write-Host 'Installation cancelled!'
        }       
    }ElseIf ($Remove.IsPresent)
    {
        Write-Host 'Passfilt removal is starting...' -ForegroundColor Cyan

        $YesOrNo = Read-Host -Prompt 'You are about to remove Svalinn password filter, please confirm (Y/N)'
        $DLLName = Read-Host -Prompt 'Enter compiled svalinn DLL name without extension (ex: svalinn64)'

        If ("y", "Y" -contains $YesOrNo){

            $LSAReg = ((Get-ItemPropertyValue -Path 'HKLM:\SYSTEM\CurrentControlSet\Control\Lsa\' -Name 'Notification Packages') | Out-String)

            $LSARegNew = $LSAReg.Replace($DLLName,'')

            Set-ItemProperty -Path 'HKLM:\SYSTEM\CurrentControlSet\Control\Lsa\' -Name "Notification Packages" -Value $LSARegNew -Type MultiString -Force

            Remove-Item 'HKLM:\SOFTWARE\passfilt' -Recurse -Force

            Write-Host 'Svalinn LSA registry key and all config keys have been removed...' -ForegroundColor Cyan

            Restart-Computer -Force -Confirm:$true
        }
        else{
            Write-Host 'Removal cancelled!'
        }
    }ElseIf ($Configure.IsPresent)
    {
        Write-Host 'Configuration is starting...' -ForegroundColor Cyan

        $YesOrNo = Read-Host -Prompt 'You are about to configure Svalinn password filter, please confirm (Y/N)'

        If ("y", "Y" -contains $YesOrNo){

            $ConfigRegPath = Test-Path -Path 'HKLM:\SOFTWARE\passfilt'

            $ServerReg = Read-Host -Prompt 'Enter fully qualified passfilt server name, or IP address'
            $PortReg = Read-Host -Prompt 'Enter port number to use for passfilt server queries'
            $EnableTLSReg = Read-Host -Prompt 'Enable TLS? Y/N (Choosing "Y" is STRONGLY recommended)'
            $DisableTLSReg = Read-Host -Prompt 'Disable TLS Validation? Y/N (Choosing "Y" is STRONGLY DISCOURAGED)'

            If ($ConfigRegPath -eq $true){
                If ($ServerReg) {
                    Set-ItemProperty -Path 'HKLM:\SOFTWARE\passfilt' -Name "Server" -Value $ServerReg -Type String -Force
                }

                If ($PortReg) {
                    Set-ItemProperty -Path 'HKLM:\SOFTWARE\passfilt' -Name "Port" -Value $PortReg -Type DWord -Force
                }

                If ("", " ", $null, "y", "Y" -contains $EnableTLSReg) {
                    Set-ItemProperty -Path 'HKLM:\SOFTWARE\passfilt' -Name "Enable TLS" -Value "1" -Type DWord -Force
                }
                else {
                    Set-ItemProperty -Path 'HKLM:\SOFTWARE\passfilt' -Name "Enable TLS" -Value "0" -Type DWord -Force
                }

                If ("", " ", $null, "n", "N" -contains $DisableTLSReg) {
                    Set-ItemProperty -Path 'HKLM:\SOFTWARE\passfilt' -Name "Disable TLS Validation" -Value "0" -Type DWord -Force
                }
                else {
                    Set-ItemProperty -Path 'HKLM:\SOFTWARE\passfilt' -Name "Disable TLS Validation" -Value "1" -Type DWord -Force
                }
            }
            else{
                New-Item -Path 'HKLM:\SOFTWARE\passfilt' -Force | Out-Null
                
                If ($ServerReg) {
                    New-ItemProperty -Path 'HKLM:\SOFTWARE\passfilt' -Name "Server" -Value $ServerReg -PropertyType String -Force
                }

                If ($PortReg) {
                    New-ItemProperty -Path 'HKLM:\SOFTWARE\passfilt' -Name "Port" -Value $PortReg -PropertyType DWord -Force
                }

                If ("", " ", $null, "y", "Y" -contains $EnableTLSReg) {
                    New-ItemProperty -Path 'HKLM:\SOFTWARE\passfilt' -Name "Enable TLS" -Value "1" -PropertyType DWord -Force
                }
                else {
                    New-ItemProperty -Path 'HKLM:\SOFTWARE\passfilt' -Name "Enable TLS" -Value "0" -PropertyType DWord -Force
                }

                If ("", " ", $null, "n", "N" -contains $DisableTLSReg) {
                    New-ItemProperty -Path 'HKLM:\SOFTWARE\passfilt' -Name "Disable TLS Validation" -Value "0" -PropertyType DWord -Force
                }
                else {
                    New-ItemProperty -Path 'HKLM:\SOFTWARE\passfilt' -Name "Disable TLS Validation" -Value "1" -PropertyType DWord -Force
                }
            }
            Write-Host 'Configuration registry keys are entered...' -ForegroundColor Cyan

            Write-Host 'Configuration complete, a reboot is recommended! Check svalinn status with option "-Status"' -ForegroundColor Cyan

            Restart-Computer -Force -Confirm:$true
        }
        else{
            Write-Host 'Configuration cancelled!'
        }
    }ElseIf ($Status.IsPresent)
    {
        Write-Host 'Gathering svalinn status report...please wait one moment' -ForegroundColor Cyan

        $ErrorActionPreference = 'ignore'
        $ModLoad = gwmi -namespace root\cimv2 -class CIM_ProcessExecutable | % {[wmi]"$($_.Antecedent)" | select * } | ?{$_.FileName -like "svalinn*"} | select Name
        $ErrorActionPreference = 'continue'

        $EventLoad = Get-EventLog -Log "Application" -Source "svalinn" -Newest 5 | select Source,@{Label='Event';Expression={($_.ReplacementStrings)}}
        
        $ConfigRegPath = Test-Path -Path 'HKLM:\SOFTWARE\passfilt' -ErrorAction SilentlyContinue
        $ConfigRegKeys = Get-ItemProperty -Path 'HKLM:\SOFTWARE\passfilt' -ErrorAction SilentlyContinue

        $StatusObject = New-Object -TypeName psobject -Property @{
            DLLModuleStatus = if ($ModLoad){($ModLoad + ' module loaded')}else{'Svalinn DLL not currently active, verify activity in event logs'}
            SvalinnEventLog = if ($EventLoad){($EventLoad)}else{'No recent Svalinn activity found in event logs'}
            ConfigRegPath = if ($ConfigRegPath -eq $false){'Config registry path does not exist'}else{'Exists'}
            ConfigRegKeys = if (!$ConfigRegKeys){'Config registry keys do not exist'}else{$ConfigRegKeys | `
                select @{Label='Server';Expression={$_.Server.ToString()}},`
                @{Label='Port';Expression={$_.Port.ToString()}},`
                @{Label='EnableTLS';Expression={$_."Enable TLS".ToString()}},`
                @{Label='DisableTLSValidation';Expression={$_."Disable TLS Validation".ToString()}}}
        }
        $StatusObject | select DLLModuleStatus,SvalinnEventLog,ConfigRegKeys,ConfigRegPath | fl
    }
    ElseIf (-not ($Install.IsPresent -or $Remove.IsPresent -or $Configure.IsPresent -or $Status.IsPresent))
    {
        Write-Host 'You need to specify a parameter (-Install, -Remove, -Configure, -Status)'
    }
}