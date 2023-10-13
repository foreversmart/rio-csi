package lvm

var (
	Enums = map[string][]string{
		LVPermissions:      {"unknown", "writeable", "read-only", "read-only-override"},
		LVWhenFull:         {"error", "queue"},
		RaidSyncAction:     {"idle", "frozen", "resync", "recover", "check", "repair"},
		LVHealthStatus:     {"", "partial", "refresh needed", "mismatches exist"},
		VGAllocationPolicy: {"normal", "contiguous", "cling", "anywhere", "inherited"},
		VGPermissions:      {"writeable", "read-only"},
	}
)

const (
	// MinExtentRoundOffSize represents minimum size (256Mi) to roundoff the volume
	// group size in case of thin pool provisioning
	MinExtentRoundOffSize = 268435456

	// BlockCleanerCommand is the command used to clean filesystem on the device
	BlockCleanerCommand = "wipefs"

	// YES is ThinProvision true
	YES        = "yes"
	LVThinPool = "thin-pool"
)

// lvm command related constants
const (
	VGCreate = "vgcreate"
	VGList   = "vgs"

	LVCreate = "lvcreate"
	LVRemove = "lvremove"
	LVExtend = "lvextend"
	LVList   = "lvs"

	PVList = "pvs"
	PVScan = "pvscan"
)

// lvm vg, lv & pv fields related constants
const (
	VGName              = "vg_name"
	VGUUID              = "vg_uuid"
	VGPVvount           = "pv_count"
	VGLvCount           = "lv_count"
	VGMaxLv             = "max_lv"
	VGMaxPv             = "max_pv"
	VGSnapCount         = "snap_count"
	VGMissingPvCount    = "vg_missing_pv_count"
	VGMetadataCount     = "vg_mda_count"
	VGMetadataUsedCount = "vg_mda_used_count"
	VGSize              = "vg_size"
	VGFreeSize          = "vg_free"
	VGMetadataSize      = "vg_mda_size"
	VGMetadataFreeSize  = "vg_mda_free"
	VGPermissions       = "vg_permissions"
	VGAllocationPolicy  = "vg_allocation_policy"

	LVName            = "lv_name"
	LVFullName        = "lv_full_name"
	LVUUID            = "lv_uuid"
	LVPath            = "lv_path"
	LVDmPath          = "lv_dm_path"
	LVActive          = "lv_active"
	LVSize            = "lv_size"
	LVMetadataSize    = "lv_metadata_size"
	LVSegtype         = "segtype"
	LVHost            = "lv_host"
	LVPool            = "pool_lv"
	LVPermissions     = "lv_permissions"
	LVWhenFull        = "lv_when_full"
	LVHealthStatus    = "lv_health_status"
	RaidSyncAction    = "raid_sync_action"
	LVDataPercent     = "data_percent"
	LVMetadataPercent = "metadata_percent"
	LVSnapPercent     = "snap_percent"

	PVName             = "pv_name"
	PVUUID             = "pv_uuid"
	PVInUse            = "pv_in_use"
	PVAllocatable      = "pv_allocatable"
	PVMissing          = "pv_missing"
	PVSize             = "pv_size"
	PVFreeSize         = "pv_free"
	PVUsedSize         = "pv_used"
	PVMetadataSize     = "pv_mda_size"
	PVMetadataFreeSize = "pv_mda_free"
	PVDeviceSize       = "dev_size"
)
