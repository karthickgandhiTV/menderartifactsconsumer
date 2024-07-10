package api

import "sync"

type Config struct {
	API APIConfig
}

type APIConfig struct {
	AuthLogin                string
	V2uriDevices             string
	V2uriDevicesCount        string
	V2uriDevicesSearch       string
	V2uriDevice              string
	V2uriDeviceAuthSet       string
	V2uriDeviceAuthSetStatus string
	V2uriToken               string
	V2uriDevicesLimit        string
}

var instance *Config
var once sync.Once

func GetConfig() *Config {
	once.Do(func() {
		instance = &Config{
			API: APIConfig{
				AuthLogin:                "/api/management/v1/useradm/auth/login",
				V2uriDevices:             "/api/management/v2/devauth/devices",
				V2uriDevicesCount:        "/api/management/v2/devauth/devices/count",
				V2uriDevicesSearch:       "/api/management/v2/devauth/devices/search",
				V2uriDevice:              "/api/management/v2/devauth/devices/#id",
				V2uriDeviceAuthSet:       "/api/management/v2/devauth/devices/#id/auth/#aid",
				V2uriDeviceAuthSetStatus: "/api/management/v2/devauth/devices/#id/auth/#aid/status",
				V2uriToken:               "/api/management/v2/devauth/tokens/#id",
				V2uriDevicesLimit:        "/api/management/v2/devauth/limits/#name",
			},
		}
	})
	return instance
}
