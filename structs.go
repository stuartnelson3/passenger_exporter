package main

type Info struct {
	CapacityUsed            string       `xml:"capacity_used"`
	MaxProcessCount         string       `xml:"max"`
	PassengerVersion        string       `xml:"passenger_version"`
	AppCount                string       `xml:"group_count"`
	TopLevelRequestsInQueue string       `xml:"get_wait_list_size"`
	CurrentProcessCount     string       `xml:"process_count"`
	SuperGroups             []SuperGroup `xml:"supergroups>supergroup"`
}

type SuperGroup struct {
	RequestsInQueue string `xml:"get_wait_list_size"`
	CapacityUsed    string `xml:"capacity_used"`
	State           string `xml:"state"`
	Group           Group  `xml:"group"`
	Name            string `xml:"name"`
}

type Group struct {
	Environment           string    `xml:"environment"`
	DisabledProcessCount  string    `xml:"disabled_process_count"`
	UID                   string    `xml:"uid"`
	GetWaitListSize       string    `xml:"get_wait_list_size"`
	CapacityUsed          string    `xml:"capacity_used"`
	Name                  string    `xml:"name"`
	AppType               string    `xml:"app_type"`
	AppRoot               string    `xml:"app_root"`
	User                  string    `xml:"user"`
	ComponentName         string    `xml:"component_name"`
	LifeStatus            string    `xml:"life_status"`
	UUID                  string    `xml:"uuid"`
	Default               string    `xml:"default,attr"`
	DisablingProcessCount string    `xml:"disabling_process_count"`
	EnabledProcessCount   string    `xml:"enabled_process_count"`
	DisableWaitListSize   string    `xml:"disable_wait_list_size"`
	GID                   string    `xml:"gid"`
	ProcessesSpawning     string    `xml:"processes_being_spawned"`
	Options               Options   `xml:"options"`
	Processes             []Process `xml:"processes>process"`
}

type Process struct {
	CodeRevision        string `xml:"code_revision"`
	Enabled             string `xml:"enabled"`
	SpawnEndTime        string `xml:"spawn_end_time"`
	HasMetrics          string `xml:"has_metrics"`
	LifeStatus          string `xml:"life_status"`
	Busyness            string `xml:"busyness"`
	RealMemory          string `xml:"real_memory"`
	StickySessionID     string `xml:"sticky_session_id"`
	PSS                 string `xml:"pss"`
	Command             string `xml:"command"`
	LastUsed            string `xml:"last_used"`
	CPU                 string `xml:"cpu"`
	SpawnerCreationTime string `xml:"spawner_creation_time"`
	LastUsedDesc        string `xml:"last_used_desc"`
	Uptime              string `xml:"uptime"`
	Swap                string `xml:"swap"`
	Sessions            string `xml:"sessions"`
	RSS                 string `xml:"rss"`
	PrivateDirty        string `xml:"private_dirty"`
	RequestsProcessed   string `xml:"processed"`
	ProcessGroupID      string `xml:"process_group_id"`
	PID                 string `xml:"pid"`
	GUPID               string `xml:"gupid"`
	VMSize              string `xml:"vmsize"`
	Concurrency         string `xml:"concurrency"`
	SpawnStartTime      string `xml:"spawn_start_time"`

	// Processes are restarted at an offset, user-defined interval. The
	// restarted process is appended to the end of the status output.  For
	// maintaining consistent process identifiers between process starts,
	// pids are mapped to an identifier based on process count. When a new
	// process/pid appears, it is mapped to either the first empty place
	// within the global map storing process identifiers, or mapped to
	// pid:id pair in the map.
	BucketID int
}

type Options struct {
	DefaultGroup              string `xml:"default_group"`
	RubyBinPath               string `xml:"ruby"`
	USTRouterAddress          string `xml:"ust_router_address"`
	USTRouterPassword         string `xml:"ust_router_password"`
	StartCommand              string `xml:"start_command"`
	USTRouterUsername         string `xml:"ust_router_username"`
	MaxPreloaderIdleTime      string `xml:"max_preloader_idle_time"`
	BaseURI                   string `xml:"base_uri"`
	SpawnMethod               string `xml:"spawn_method"`
	AppType                   string `xml:"app_type"`
	Environment               string `xml:"environment"`
	Analytics                 string `xml:"analytics"`
	MinProcesses              string `xml:"min_processes"`
	StartTimeout              string `xml:"start_timeout"`
	AppRoot                   string `xml:"app_root"`
	ProcessTitle              string `xml:"process_title"`
	Debugger                  string `xml:"debugger"`
	DefaultUser               string `xml:"default_user"`
	MaxOutOfBandWorkInstances string `xml:"max_out_of_band_work_instances"`
	MaxProcesses              string `xml:"max_processes"`
	AppGroupName              string `xml:"app_group_name"`
	StartupFile               string `xml:"startup_file"`
	IntegrationMode           string `xml:"integration_mode"`
	LogLevel                  string `xml:"log_level"`
}
