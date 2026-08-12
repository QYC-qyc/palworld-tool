package tool

import (
	"encoding/base64"

	"github.com/spf13/viper"
	"paladmin/internal/executor"
	"paladmin/internal/logger"
)

// executeCommand 建立 RCON 连接并执行单条命令，支持 Base64（PalGuard/中文）
func executeCommand(command string) (*executor.Executor, string, error) {
	useBase64 := viper.GetBool("rcon.use_base64")

	exec, err := executor.NewExecutor(
		viper.GetString("rcon.address"),
		viper.GetString("rcon.password"),
		viper.GetInt("rcon.timeout"),
		true,
	)
	if err != nil {
		return nil, "", err
	}

	if useBase64 {
		command = base64.StdEncoding.EncodeToString([]byte(command))
	}

	response, err := exec.Execute(command)
	if err != nil {
		return nil, "", err
	}

	if useBase64 {
		decoded, err := base64.StdEncoding.DecodeString(response)
		if err != nil {
			logger.Warnf("base64 解码失败: %v", err)
			return exec, response, nil
		}
		response = string(decoded)
	}
	return exec, response, nil
}

// CustomCommand 执行任意 RCON 命令
func CustomCommand(command string) (string, error) {
	exec, response, err := executeCommand(command)
	if exec != nil {
		defer exec.Close()
	}
	if err != nil {
		return "", err
	}
	return response, nil
}
