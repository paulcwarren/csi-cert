package csi_cert

import (
	"encoding/json"
	"io/ioutil"
	"os/user"
	"path/filepath"
)

type CertificationFixture struct {
	ControllerAddress string `json:"controller_address"`
	NodeAddress       string `json:"node_address"`
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

	usr, err := user.Current()
	if certificationFixture.ControllerAddress[:2] == "~/" {
		certificationFixture.ControllerAddress = filepath.Join(usr.HomeDir, certificationFixture.ControllerAddress[2:])
	}
	if certificationFixture.NodeAddress[:2] == "~/" {
		certificationFixture.NodeAddress = filepath.Join(usr.HomeDir, certificationFixture.NodeAddress[2:])
	}

	if err != nil {
		return CertificationFixture{}, err
	}

	return certificationFixture, nil
}
