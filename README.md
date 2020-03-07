# go-file-retry
Utility to move files from failed folder to working folder, with retry limit.

After the retry limit is reached, the files will be moved to a deadletter queue.

Below are the possible configuration override values and their default value, override file name is "go-file-retry.json"
```json
{
	"ServerName": "os.ServerName",
	"SrcDir": "\\retry\\",
	"DstDir": "\\work\\",
	"DeadLtrDir": "\\deadletter\\",
	"AppName": "Go - FileRetry",
	"AppVer": "1.0",
	"SysLogSrv": "localhost",
	"SysLogPort": "514",
	"LogLevel": "info",
	"RetryCount": 1,
	"RetryDelay": 15
}
```

Default config does account for Windows vs Linux/Unix, so the directory paths are the same but with the appropriate path seperators and the executioun location will be used for the base of the path.  

