package main

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api/v1"
)

const createdField string = "created"
const expiresField string = "expires"
const createdLayout string = "20060102150405"

type nameSpaceExpired struct {
	Name          string
	ExpiredTime   string
	CurrentTime   string
	GivenTime     string
	GivenTimeConv string
}

func (args expiredNSArgs) execute() {
	// run command
	getExpiredNS(args.clientset)
}

func getExpiredNS(clientset *kubernetes.Clientset) {
	// get namespaces
	ns := clientset.CoreV1Client.Namespaces()
	nsList, err := ns.List(v1.ListOptions{})
	if err != nil {
		panic(err)
	}

	// iterate over all namespaces and get the annotatios
	expiredNamespaces := []*nameSpaceExpired{}
	for _, nsObj := range nsList.Items {
		// get annotations from namespace
		nsAnno := nsObj.Annotations

		// iterate annotations
		var createdTime time.Time
		var createdTimeFound bool = false
		var expiresFieldValue string
		var expiredNamespace nameSpaceExpired = nameSpaceExpired{}
		for id, anno := range nsAnno {
			// did we found the created and expires tag?
			switch id {
			case createdField:
				t, err := time.ParseInLocation(createdLayout, anno, time.Local)
				if err != nil {
					fmt.Println(err)
				} else {
					expiredNamespace.GivenTime = anno
					createdTime = t
					createdTimeFound = true
				}
			case expiresField:
				expiresFieldValue = anno
			}
		}

		if expiresFieldValue != "" && createdTimeFound && expiresFieldValue != "none" {
			expiredTime := calculateExpireDate(createdTime, expiresFieldValue)
			if time.Now().Local().After(expiredTime) {
				expiredNamespace.Name = nsObj.ObjectMeta.Name
				expiredNamespace.ExpiredTime = expiredTime.String()
				expiredNamespace.CurrentTime = time.Now().Local().String()
				expiredNamespace.GivenTimeConv = createdTime.String()
				expiredNamespaces = append(expiredNamespaces, &expiredNamespace)
			}
		}
	}

	// print out all expired namespaces as json
	b, err := json.Marshal(expiredNamespaces)
	if err != nil {
		panic(err)
	}
	fmt.Println(string(b))
}

func calculateExpireDate(t time.Time, addTime string) time.Time {
	// get the value of time (e.g. 12)
	i, err := strconv.Atoi(addTime[0 : len(addTime)-1])
	if err != nil {
		fmt.Println(err)
		return t
	}

	// get the time type (e.g. d for days)
	givenType := strings.ToLower(addTime[len(addTime)-1:])

	// find out the type
	var timeType time.Duration
	switch givenType {
	case "s":
		timeType = time.Second
	case "m":
		timeType = time.Minute
	case "h":
		timeType = time.Hour
	case "d":
		return t.AddDate(0, 0, i)
	case "w":
		return t.AddDate(0, 0, i*7)
	default:
		return t
	}

	// calculate new expired time
	t = t.Add(timeType * time.Duration(i))
	return t
}