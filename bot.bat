@echo off
setlocal enabledelayedexpansion

REM WeChat Work Bot Windows Batch Script

REM --- Configuration ---
set "SERVER_URL={SERVER_URL_Template}"
set "DEFAULT_BOT_ID={BOT_ID_Template}"
set "SECURITY_CODE={SECURITY_CODE_Template}"

REM --- Usage ---
:usage
echo.
echo Usage: %~n0 ^<command^> ^<argument^>
echo.
echo Commands:
echo.
echo   send     ^<message^>      Send text as Markdown message.
echo.
echo   sendfile ^<file_path^>     Send a file.
echo.
echo Examples:
echo.
echo   %~n0 send "### Warning\nPlease check system status."
echo.
echo   %~n0 sendfile API.md
echo.
echo Default Bot ID: %DEFAULT_BOT_ID%
echo Default Security Code: %SECURITY_CODE%
goto :eof

REM --- Parameter Check ---
if "%~1"=="" (
    echo Error: No command provided.
    call :usage
    exit /b 1
)
if "%~2"=="" (
    echo Error: No argument provided.
    call :usage
    exit /b 1
)

set "COMMAND=%~1"
set "ARGUMENT=%~2"

REM --- Command Dispatch ---
if /i "%COMMAND%"=="send" goto :cmd_send
if /i "%COMMAND%"=="sendfile" goto :cmd_sendfile

echo Error: Unknown command '%COMMAND%'
call :usage
exit /b 1

REM --- Send Message ---
:cmd_send
echo Sending Markdown message to Bot ID: %DEFAULT_BOT_ID%...

REM Escape JSON special characters
set "JSON_CONTENT=%ARGUMENT%"
set "JSON_CONTENT=!JSON_CONTENT:"=\"!"
set "JSON_CONTENT=!JSON_CONTENT:\=\\!"

REM Create temporary JSON file
set "TEMP_JSON=%TEMP%\qywxbot_payload.json"
echo { > "%TEMP_JSON%"
echo   "id": %DEFAULT_BOT_ID%, >> "%TEMP_JSON%"
echo   "security_code": "%SECURITY_CODE%", >> "%TEMP_JSON%"
echo   "msgtype": "markdown", >> "%TEMP_JSON%"
echo   "content": "!JSON_CONTENT!" >> "%TEMP_JSON%"
echo } >> "%TEMP_JSON%"

REM Send request
curl -s -X POST "%SERVER_URL%/send" -H "Content-Type: application/json" -d @"%TEMP_JSON%"

REM Clean up
del "%TEMP_JSON%" 2>nul

echo.
echo Message sent successfully.
goto :eof

REM --- Send File ---
:cmd_sendfile
if not exist "%ARGUMENT%" (
    echo Error: File '%ARGUMENT%' not found.
    exit /b 1
)
echo Sending file '%ARGUMENT%' to Bot ID: %DEFAULT_BOT_ID%...
curl -s -X POST "%SERVER_URL%/sendfile" -F "id=%DEFAULT_BOT_ID%" -F "security_code=%SECURITY_CODE%" -F "media=@%ARGUMENT%"
echo.
echo File sent successfully.
goto :eof