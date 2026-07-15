@echo off
set PATH=%PATH%;C:\Program Files\Go\bin
echo Building radio.exe...
go build -ldflags="-s -w -H windowsgui" -o radio.exe .
if %ERRORLEVEL% NEQ 0 (
    echo Build failed!
    pause
    exit /b %ERRORLEVEL%
)
echo Done: %~dp0radio.exe
echo.
echo Copy to apps? (y/n)
choice /c yn /n
if %ERRORLEVEL% EQU 2 exit /b 0
copy /Y radio.exe "C:\Apps\radio\radio.exe"
echo Copied to C:\Apps\radio\radio.exe
pause
