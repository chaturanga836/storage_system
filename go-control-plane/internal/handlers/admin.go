package handlers

import (
    "bufio"
    "encoding/json"
    "net/http"
    "os"
)

type AuditLogEntry struct {
    Timestamp   string `json:"timestamp"`
    Action      string `json:"action"`
    PerformedBy string `json:"performed_by"`
    TargetUser  string `json:"target_user"`
    Details     string `json:"details"`
}

func GetAuditLogs(w http.ResponseWriter, r *http.Request) {
    file, err := os.Open("logs/audit.jsonl")
    if err != nil {
        http.Error(w, "Failed to open audit log", http.StatusInternalServerError)
        return
    }
    defer file.Close()

    var logs []AuditLogEntry
    scanner := bufio.NewScanner(file)
    for scanner.Scan() {
        var entry AuditLogEntry
        if err := json.Unmarshal(scanner.Bytes(), &entry); err == nil {
            logs = append(logs, entry)
        }
    }

    if err := scanner.Err(); err != nil {
        http.Error(w, "Error reading audit log", http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(logs)
}
