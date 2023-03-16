package iscsi

import "sync"

var (
	openIscsiDir = "cd /iscsi"
	openBlockDir = "cd /backstores/block"

	// must under target dir
	openAclsDir = "cd tpg1/acls"
	openLunsDir = "cd tpg1/luns"

	lsCmd = "ls"

	createCmd      = "create %s"
	deleteCmd      = "delete %s"
	createBlockCmd = "create %s %s"
	cdCmd          = "cd %s"
	setUserIDCmd   = "set auth userid=%s"
	setPasswordCmd = "set auth password=%s"

	setDiscoveryAuth = "set discovery_auth enable=1 userid=%s password=%s"

	exitCmd = "exit\n"
)

var (
	Lock sync.Mutex
)
