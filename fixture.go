package csi_cert

import (
	"encoding/json"
	"io/ioutil"
	"os/user"
	"path/filepath"
)

type CertificationFixture struct {
	VolmanDriverPath string                  `json:"volman_driver_path"`
	DriverAddress    string                  `json:"driver_address"`
	DriverName       string                  `json:"driver_name"`
	CreateConfig     voldriver.CreateRequest `json:"create_config"`
	TLSConfig        *voldriver.TLSConfig    `json:"tls_config,omitempty"`
}

func LoadCertificationFixture(fileName string) (CertificationFixture, error) {
	bytes, err := ioutil.ReadFile(fileName)
	if err != nil {
		return CertificationFixture{}, err
	}

	certificationFixture := CertificationFixture{}
	err = json.Unmarshal(bytes, &certificationFixture)
	if err != nil {
		return CertificationFixture{}, err
	}

	// make sure that the paths are absolute
	usr, err := user.Current()
	if certificationFixture.VolmanDriverPath[:2] == "~/" {
		if err != nil {
			return CertificationFixture{}, err
		}
		certificationFixture.VolmanDriverPath = filepath.Join(usr.HomeDir, certificationFixture.VolmanDriverPath[2:])
	}
	if certificationFixture.DriverAddress[:2] == "~/" {
		certificationFixture.DriverAddress = filepath.Join(usr.HomeDir, certificationFixture.DriverAddress[2:])
	}
	if certificationFixture.TLSConfig != nil {
		if certificationFixture.TLSConfig.CAFile[:2] == "~/" {
			certificationFixture.TLSConfig.CAFile = filepath.Join(usr.HomeDir, certificationFixture.TLSConfig.CAFile[2:])
		}
		if certificationFixture.TLSConfig.CertFile[:2] == "~/" {
			certificationFixture.TLSConfig.CertFile = filepath.Join(usr.HomeDir, certificationFixture.TLSConfig.CertFile[2:])
		}
		if certificationFixture.TLSConfig.KeyFile[:2] == "~/" {
			certificationFixture.TLSConfig.KeyFile = filepath.Join(usr.HomeDir, certificationFixture.TLSConfig.KeyFile[2:])
		}
	}
	certificationFixture.VolmanDriverPath, err = filepath.Abs(certificationFixture.VolmanDriverPath)
	if err != nil {
		return CertificationFixture{}, err
	}

	return certificationFixture, nil
}
