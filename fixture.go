package csi_cert

import (
	"encoding/json"
	"io/ioutil"
	"os/user"
	"path/filepath"
)

type CertificationFixture struct {
	DriverAddress string `json:"driver_address"`
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
	if certificationFixture.DriverAddress[:2] == "~/" {
		certificationFixture.DriverAddress = filepath.Join(usr.HomeDir, certificationFixture.DriverAddress[2:])
	}

	if err != nil {
		return CertificationFixture{}, err
	}

	return certificationFixture, nil
}
