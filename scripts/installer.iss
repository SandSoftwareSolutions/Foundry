; Inno Setup script for a per-user Windows installer. Build with:
;   iscc /DAppVersion=1.2.3 /DSourceExe=..\dist\foundry.exe scripts\installer.iss
#ifndef AppVersion
  #define AppVersion "0.0.0"
#endif
#ifndef SourceExe
  #define SourceExe "..\foundry.exe"
#endif

[Setup]
AppName=Foundry
AppVersion={#AppVersion}
AppPublisher=Sand Software Solutions
DefaultDirName={localappdata}\Programs\Foundry
DisableProgramGroupPage=yes
PrivilegesRequired=lowest
OutputBaseFilename=foundry-{#AppVersion}-setup
Compression=lzma
SolidCompression=yes
ChangesEnvironment=yes
LicenseFile=..\LICENSE

[Files]
Source: "{#SourceExe}"; DestDir: "{app}"; DestName: "foundry.exe"; Flags: ignoreversion

[Registry]
; Add the install directory to the user's PATH (not the machine's), once.
Root: HKCU; Subkey: "Environment"; ValueType: expandsz; ValueName: "Path"; \
  ValueData: "{olddata};{app}"; Check: NeedsAddPath('{app}')

[Code]
function NeedsAddPath(Param: string): Boolean;
var
  OrigPath: string;
begin
  if not RegQueryStringValue(HKEY_CURRENT_USER, 'Environment', 'Path', OrigPath) then
  begin
    Result := True;
    Exit;
  end;
  Result := Pos(';' + Lowercase(Param) + ';', ';' + Lowercase(OrigPath) + ';') = 0;
end;
