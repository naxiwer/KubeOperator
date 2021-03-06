package model

import (
	"fmt"
	"github.com/KubeOperator/KubeOperator/pkg/db"
	"github.com/KubeOperator/KubeOperator/pkg/model/common"
	_ "github.com/KubeOperator/KubeOperator/pkg/service/cluster/adm/facts"
	"github.com/KubeOperator/KubeOperator/pkg/util/ssh"
	"github.com/KubeOperator/kobe/api"
	uuid "github.com/satori/go.uuid"
	"time"
)

type ClusterNode struct {
	common.BaseModel
	ID        string `json:"-"`
	Name      string `json:"name"`
	HostID    string `json:"-"`
	Host      Host   `json:"-" gorm:"save_associations:false"`
	ClusterID string `json:"clusterId"`
	Role      string `json:"role"`
	Status    string `json:"status"`
	Dirty     bool   `json:"dirty"`
	Message   string `json:"message"`
}

type Registry struct {
	Architecture     string
	RegistryProtocol string
	RegistryHostname string
}

func (n *ClusterNode) BeforeCreate() (err error) {
	n.ID = uuid.NewV4().String()
	return nil
}

func (n ClusterNode) GetRegistry(arch string) (*Registry, error) {
	var systemRegistry SystemRegistry
	var systemSetting SystemSetting
	var registry Registry

	err := db.DB.Where(&SystemSetting{Key: "arch_type"}).First(&systemSetting).Error
	if err != nil {
		return nil, err
	}
	if systemSetting.Value == "single" {
		err = db.DB.Where(&SystemSetting{Key: "ip"}).First(&systemSetting).Error
		if err != nil {
			return nil, err
		}
		registry.RegistryHostname = systemSetting.Value
		err = db.DB.Where(&SystemSetting{Key: "REGISTRY_PROTOCOL"}).First(&systemSetting).Error
		if err != nil {
			return nil, err
		}
		registry.RegistryProtocol = systemSetting.Value
		switch n.Host.Architecture {
		case "x86_64":
			registry.Architecture = "amd64"
		case "aarch64":
			registry.Architecture = "arm64"
		default:
			registry.Architecture = "amd64"
		}
	} else if systemSetting.Value == "mixed" {
		err := db.DB.Where(&SystemRegistry{Architecture: arch}).First(&systemRegistry).Error
		if err != nil {
			return nil, err
		}
		registry.RegistryHostname = systemRegistry.RegistryHostname
		registry.RegistryProtocol = systemRegistry.RegistryProtocol
		switch n.Host.Architecture {
		case "x86_64":
			registry.Architecture = "amd64"
		case "aarch64":
			registry.Architecture = "arm64"
		}
	}
	fmt.Println(registry)
	return &registry, nil
}

func (n ClusterNode) ToKobeHost() *api.Host {
	password, privateKey, _ := n.Host.GetHostPasswordAndPrivateKey()
	r, _ := n.GetRegistry(n.Host.Architecture)
	return &api.Host{
		Ip:         n.Host.Ip,
		Name:       n.Name,
		Port:       int32(n.Host.Port),
		User:       n.Host.Credential.Username,
		Password:   password,
		PrivateKey: string(privateKey),
		Vars: map[string]string{
			"has_gpu":           fmt.Sprintf("%v", n.Host.HasGpu),
			"architecture":      r.Architecture,
			"registry_protocol": r.RegistryProtocol,
			"registry_hostname": r.RegistryHostname,
		},
	}
}

func (n ClusterNode) ToSSHConfig() ssh.Config {
	password, privateKey, _ := n.Host.GetHostPasswordAndPrivateKey()
	return ssh.Config{
		User:        n.Host.Credential.Username,
		Host:        n.Host.Ip,
		Port:        n.Host.Port,
		PrivateKey:  privateKey,
		Password:    password,
		DialTimeOut: 5 * time.Second,
		Retry:       3,
	}
}
