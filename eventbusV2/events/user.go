package events

// ===== 一般事件 (Fire-and-forget) =====



type StationStatus struct {
    StationID string
    Status    string 
    Error     string
    Other     string 
}



type QamsCommand struct {
    StationId string
    Cmd string
}