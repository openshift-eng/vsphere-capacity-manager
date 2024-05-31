package utils

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/vapi/rest"
	"github.com/vmware/govmomi/vim25"
	"github.com/vmware/govmomi/vim25/soap"
)

// ClientLogout is empty function that logs out of vSphere clients
type ClientLogout func()

// CreateVSphereClients creates the SOAP and REST client to access
// different portions of the vSphere API
// e.g. tags are only available in REST
func CreateVSphereClients(ctx context.Context, vcenter, username, password string) (*vim25.Client, *rest.Client, ClientLogout, error) {
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	u, err := soap.ParseURL(vcenter)
	if err != nil {
		return nil, nil, nil, err
	}
	u.User = url.UserPassword(username, password)
	c, err := govmomi.NewClient(ctx, u, false)

	if err != nil {
		return nil, nil, nil, err
	}

	restClient := rest.NewClient(c.Client)
	err = restClient.Login(ctx, u.User)
	if err != nil {
		logoutErr := c.Logout(context.TODO())
		if logoutErr != nil {
			err = logoutErr
		}
		return nil, nil, nil, err
	}

	return c.Client, restClient, func() {
		c.Logout(context.TODO())
		restClient.Logout(context.TODO())
	}, nil
}

func LoadCredentialsFromPath(path string) (map[string]string, error) {
	credentials := make(map[string]string)
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("error opening file: %v", err)
	}

	contentStr := string(content)

	lines := strings.Split(contentStr, "\n")
	usernames := make([]string, 0)
	passwords := make([]string, 0)

	for _, line := range lines {
		parts := strings.Split(line, "=")
		if len(parts) != 2 {
			continue
		}

		// remove the parenthesis
		credPart := strings.TrimSpace(parts[1])
		credPart = credPart[1 : len(credPart)-1]
		credParts := strings.Split(credPart, " ")
		for _, credVal := range credParts {
			credVal = credVal[1 : len(credVal)-1]
			if parts[0] == "vcenter_usernames" {
				usernames = append(usernames, credVal)
			} else if parts[0] == "vcenter_passwords" {
				passwords = append(passwords, credVal)
			} else {
				log.Printf("Unrecognized credential part: %s", parts[0])
			}
		}
	}
	for idx, username := range usernames {
		credentials[username] = passwords[idx]
	}
	return credentials, nil
}
