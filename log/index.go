package log

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var Logger *zap.Logger

// InitLog 初始化 zap logger
// logFilePath: 日誌檔案路徑，例如 "app.log"


func InitLog() {
    kenmecPath, err := GetKenmecFilePath()
    if err != nil {
        panic(err)
    }

    // 日誌目錄
    logDir := filepath.Join(kenmecPath, "_logs/charge_station")
    err = os.MkdirAll(logDir, os.ModePerm)
    if err != nil {
        panic("無法建立 log 目錄: " + err.Error())
    }

    // 產生每日 log 檔案名稱
    today := time.Now().Format("2006-01-02") // YYYY-MM-DD
    logFilePath := filepath.Join(logDir, fmt.Sprintf("charge_station-%s.log", today))

    // 開啟或建立 log 檔案
    file, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0666)
    if err != nil {
        panic("無法開啟 log 檔案: " + err.Error())
    }

    // Encoder 設定
    encoderCfg := zap.NewProductionEncoderConfig()
    encoderCfg.TimeKey = "T"
    encoderCfg.LevelKey = "L"
    encoderCfg.CallerKey = "C"
    encoderCfg.EncodeTime = zapcore.ISO8601TimeEncoder
    encoderCfg.EncodeLevel = zapcore.CapitalColorLevelEncoder
    encoderCfg.EncodeCaller = zapcore.ShortCallerEncoder

    // Terminal encoder
    consoleEncoder := zapcore.NewConsoleEncoder(encoderCfg)
    // File encoder (json)
    fileEncoder := zapcore.NewJSONEncoder(encoderCfg)

    // 同時輸出到 terminal 與檔案
    core := zapcore.NewTee(
        zapcore.NewCore(consoleEncoder, zapcore.AddSync(os.Stdout), zap.DebugLevel),
        zapcore.NewCore(fileEncoder, zapcore.AddSync(file), zap.DebugLevel),
    )

    Logger = zap.New(core,  zap.AddStacktrace(zapcore.ErrorLevel))
}

func GetKenmecFilePath() (string, error) {
    currentUser, err := user.Current()
    if err != nil {
        return "", fmt.Errorf("無法獲取用戶主目錄: %w", err)
    }
    homeDir := currentUser.HomeDir
    kenmecPath := filepath.Join(homeDir, "kenmec")
    return kenmecPath, nil
}

