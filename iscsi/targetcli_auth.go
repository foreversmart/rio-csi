package iscsi

func SetDiscoveryAuth(username, password string) error {
	cmd := NewExecCmd()
	cmd.Add(openIscsiDir)
	cmd.AddFormat(setDiscoveryAuth, username, password)
	_, err := cmd.Exec()
	return err
}

// SetUpTargetAcl set target acl rules for client
func SetUpTargetAcl(target, client, username, password string) (string, error) {
	cmd := NewExecCmd()
	cmd.Add(openIscsiDir)
	cmd.AddFormat(cdCmd, target)
	cmd.Add(openAclsDir)
	// create acls
	cmd.AddFormat(createCmd, client)
	cmd.AddFormat(cdCmd, client)
	// set username and password
	cmd.AddFormat(setUserIDCmd, username)
	cmd.AddFormat(setPasswordCmd, password)
	return cmd.Exec()
}
