package main

import (
	"fmt"
	"k8s.io/helm/pkg/helm"
)

func main() {
	client := helm.NewClient(helm.Host("localhost:44134"))

	version, err := client.GetVersion()
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(version)
	list, err := client.ReleaseHistory("my-release")
	if err != nil {
		fmt.Println(err)
	}
	for _, v := range list.Releases {
		fmt.Println(v.Name)
		fmt.Println(v.Version)
		fmt.Println(v.GetInfo().Status.Code)
	}

}
