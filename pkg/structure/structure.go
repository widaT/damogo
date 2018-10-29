package structure

type FeatureFloat float32

type GroupInfo struct {
	GroupName string `json:"group_name"`
	Len       int    `json:"len"`
}

type NodeInfo struct {
	Node         string      `json:"node"`
	RegisterTime string      `json:"register_time"`
	GroupLen     int         `json:"group_len"`
	GroupInfo    []GroupInfo `json:"group_info"`
}

type HostInfo struct {
	Name         string `json:"name"`
	RegisterTime string `json:"register_time"`
}

type UserListArg struct {
	GroupId  string
	StartKey string
	EndKey   string
	Num      int
}

type MoveGroupArg struct {
	Host      string
	GroupId   string
	StartKey  string
	EndKey    string
	DestHost  string
	DestGroup string
}

type UserInfo struct {
	UID        string           `json:"uid"`
	Features   [][]FeatureFloat `json:"-"`
	GroupId    string           `json:"group_id"`
	ActionType string           `json:"-"`
	Distance   []FeatureFloat   `json:"scores"`
}

type Arg struct {
	Feature []FeatureFloat
	GroupId string
}

type ResultInfo struct {
	Distance FeatureFloat `json:"-"`
	User     UserInfo
}

type User struct {
	Id      string
	Feature []FeatureFloat
	Tag     string
}

type Result struct {
	Distance FeatureFloat
	User     User
}

type ResultSlice []Result

func (p ResultSlice) Len() int           { return len(p) }
func (p ResultSlice) Less(i, j int) bool { return p[i].Distance < p[j].Distance }
func (p ResultSlice) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }
