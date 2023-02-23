package iscsi

import "strings"

func SetDiscoveryAuth(username, password string) error {
	cmd := NewExecCmd()
	cmd.Add(openIscsiDir)
	cmd.AddFormat(setDiscoveryAuth, username, password)
	_, err := cmd.Exec()
	return err
}

// SetUpTargetAcl set target acl rules for client
func SetUpTargetAcl(target, initiator, username, password string) (string, error) {
	cmd := NewExecCmd()
	cmd.Add(openIscsiDir)
	cmd.AddFormat(cdCmd, target)
	cmd.Add(openAclsDir)
	// create acls
	cmd.AddFormat(createCmd, initiator)
	cmd.AddFormat(cdCmd, initiator)
	// set username and password
	cmd.AddFormat(setUserIDCmd, username)
	cmd.AddFormat(setPasswordCmd, password)
	res, err := cmd.Exec()
	if err != nil {
		if strings.Contains(err.Error(), "This NodeACL already exists") {
			return res, nil
		}
	}

	return res, err
}

// ListTargetAcl get target acl rules
func ListTargetAcl(target string) (aclInitiator []string, err error) {
	cmd := NewExecCmd()
	cmd.Add(openIscsiDir)
	cmd.AddFormat(cdCmd, target)
	cmd.Add(openAclsDir)
	// ls
	cmd.Add(lsCmd)
	out, err := cmd.Exec()
	if err != nil {
		return nil, err
	}

	lines := strings.Split(out, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "o- iqn.") {
			splitLoc := strings.Index(line, " .....")
			if splitLoc == -1 {
				continue
			}

			t := line[:splitLoc]
			t = strings.TrimPrefix(t, "o- ")
			aclInitiator = append(aclInitiator, t)
		}
	}

	return
}
