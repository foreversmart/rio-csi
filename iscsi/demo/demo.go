package main

import (
	"fmt"
	"qiniu.io/rio-csi/iscsi"
)

func main() {
	//fmt.Println(iscsi.CreateTarget("test", "xs2208"))
	//fmt.Println("---------------------")
	//fmt.Println(iscsi.ListTarget())
	//fmt.Println("---------------------")
	//fmt.Println(iscsi.SetUpTargetAcl("iqn.2022-11.test.srv:rio.xs2208", "qiniue", "test"))
	fmt.Println(iscsi.MountLun("iqn.2022-11.rio-csi.srv:rio.xs2298", "pvc-5446912c-8242-476e-8d46-c2745d3c47a8"))
}
