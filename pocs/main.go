package main

import (
	"fmt"
	"k8s.io/helm/pkg/helm"
)

type Deployment struct {
	Version     string `json:"version"`
	Name        string `json:"name"`
	Status      string `json:"status"`
	Description string `json:"description"`
}

func main() {
	client := helm.NewClient(helm.Host("localhost:44134"))

	_, err := client.GetVersion()
	if err != nil {
		fmt.Println(err)
	}

	s := "my-release"
	status, err := client.ReleaseStatus(s)
	if err != nil {
		fmt.Println(err)
	}

	fmt.Println("Status: ", status.Info)
	list, err := client.ReleaseHistory(s, helm.WithMaxHistory(10))
	if err != nil {
		fmt.Println(err)
	}

	fmt.Printf("->%v\n", list)
	for _, v := range list.GetReleases() {

		fmt.Printf("%v\t%v\t%v\t%v\n", v.Version, v.Name, v.GetInfo().Status.Code, v.Info.Description)

	}

}
