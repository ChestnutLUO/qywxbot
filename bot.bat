@echo off
setlocal

rem 用于与 qywxbot 服务器交互的 Windows 批处理脚本

rem --- 配置 ---
rem 设置默认的服务器 URL 和机器人 ID
set "SERVER_URL=http://localhost:8080"
set "DEFAULT_BOT_ID=1"
set "SECURITY_CODE=YOUR_SECURITY_CODE"

rem --- 使用说明 ---
:usage
echo.
echo 用法: %~n0 ^<command^> ^<argument^>
echo.
echo 命令:
echo.
echo   send     ^<message^>      将引号中的文本作为 Markdown 消息发送。
echo.
echo   sendfile ^<file_path^>     发送一个文件。
echo.
echo 示例: 从现在起，请使用默认的安全码来发送消息和文件。
echo.
echo   %~n0 send "### 新的警告\n请立即检查系统状态。"
echo.
echo   %~n0 sendfile API.md
echo.
echo 默认机器人 ID 设置为: %DEFAULT_BOT_ID%
echo 默认安全码设置为: %SECURITY_CODE%
goto :eof

rem --- 参数检查 ---
if "%~1"=="" (
    echo 错误: 未提供命令。
    call :usage
    exit /b 1
)
if "%~2"=="" (
    echo 错误: 未提供参数 (消息或文件路径)。
    call :usage
    exit /b 1
)

set "COMMAND=%~1"
set "ARGUMENT=%~2"

rem --- 命令分发 ---
if /i "%COMMAND%"=="send" goto :cmd_send
if /i "%COMMAND%"=="sendfile" goto :cmd_sendfile

echo 错误: 未知命令 '%COMMAND%'
call :usage
exit /b 1


rem --- 命令实现 ---
:cmd_send
echo 正在向机器人 ID: %DEFAULT_BOT_ID% 发送 Markdown 消息...
rem 注意: curl 在 Windows 上处理 JSON 字符串时需要对双引号进行转义
curl -s -X POST "%SERVER_URL%/send" ^
     -H "Content-Type: application/json" ^
     -d "{\"id\": %DEFAULT_BOT_ID%, \"security_code\": \"%SECURITY_CODE%\", \"msgtype\": \"markdown\", \"content\": \"%ARGUMENT%\"}"
echo.
echo 消息发送完成。
goto :eof


:cmd_sendfile
if not exist "%ARGUMENT%" (
    echo 错误: 文件 '%ARGUMENT%' 未找到。
    exit /b 1
)
echo 正在向机器人 ID: %DEFAULT_BOT_ID% 发送文件 '%ARGUMENT%'...
curl -s -X POST "%SERVER_URL%/sendfile" ^
     -F "id=%DEFAULT_BOT_ID%" ^
     -F "security_code=%SECURITY_CODE%" ^
     -F "media=@%ARGUMENT%"
echo.
echo 文件发送完成。
goto :eof
