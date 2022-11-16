package main

import (
	"fmt"
	"qiniu.io/rio-csi/iscsi"
)

func main() {
	fmt.Println(iscsi.SetUpTarget("test", "xs2208"))
	fmt.Println("---------------------")
	fmt.Println(iscsi.ListTarget())
	fmt.Println("---------------------")
	fmt.Println(iscsi.SetUpTargetAcl("iqn.2022-11.test.srv:rio.xs2208", "qiniue", "test"))
}
