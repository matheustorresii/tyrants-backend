package scene

import (
    "encoding/json"
    "log"
    "math/rand"
    "net/http"
    "sort"
    "sync"

    "github.com/gorilla/websocket"
    "github.com/matheustorresii/tyrants-back/internal/models"
)

// TyrantService defines the DB dependency we need.
type TyrantService interface {
    GetTyrant(id string) (models.Tyrant, error)
}

type Participant struct {
    Tyrant     models.Tyrant
    Enemy      bool
    FullHP     int
    CurrentHP  int
    Alive      bool
    // Attack PP tracking for this battle: attack name -> (full,current)
    AttackPP   map[string]*struct{ Full int; Current int }
}

type Client struct {
    conn *websocket.Conn
}

type Hub struct {
    mu                 sync.RWMutex
    svc                TyrantService
    clients            map[*Client]bool
    tyrantIDToClient   map[string]*Client
    participants       map[string]*Participant // key: tyrant id
    turnOrder          []string                // ordered tyrant IDs by speed desc
    turnIndex          int
    inBattle           bool
    currentActor       string
    // voting state
    votingActive       bool
    voteUntilDeath     int
    voteToParty        int
    votedAllies        map[string]bool
    totalAllies        int
}

func NewHub(svc TyrantService) *Hub {
    return &Hub{
        svc:              svc,
        clients:          make(map[*Client]bool),
        tyrantIDToClient: make(map[string]*Client),
        participants:     make(map[string]*Participant),
    }
}

var upgrader = websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}

func (h *Hub) ServeWS(w http.ResponseWriter, r *http.Request) {
    conn, err := upgrader.Upgrade(w, r, nil)
    if err != nil {
        http.Error(w, "upgrade failed", http.StatusBadRequest)
        return
    }
    client := &Client{conn: conn}
    h.mu.Lock()
    h.clients[client] = true
    h.mu.Unlock()

    // Clean up on close
    defer func() {
        h.mu.Lock()
        delete(h.clients, client)
        // remove any tyrant mapping pointing to this client
        for id, c := range h.tyrantIDToClient {
            if c == client {
                delete(h.tyrantIDToClient, id)
            }
        }
        h.mu.Unlock()
        _ = conn.Close()
    }()

    for {
        _, data, err := conn.ReadMessage()
        if err != nil {
            return
        }
        h.handleIncoming(client, data)
    }
}

// Incoming message shapes
type attackEvent struct {
    User   string `json:"user"`
    Target string `json:"target"`
    Attack string `json:"attack"`
}

type incoming struct {
    Image  *string      `json:"image,omitempty"`
    Battle *string      `json:"battle,omitempty"`
    VoteEnabled *bool   `json:"voteEnabled,omitempty"`
    Join   *string      `json:"join,omitempty"`
    Enemy  *bool        `json:"enemy,omitempty"`
    Attack *attackEvent `json:"attack,omitempty"`
    Clean  *bool        `json:"clean,omitempty"`
    Vote   *string      `json:"vote,omitempty"`
    User   *string      `json:"user,omitempty"`
}

func (h *Hub) handleIncoming(c *Client, data []byte) {
    var msg incoming
    if err := json.Unmarshal(data, &msg); err != nil {
        log.Printf("scene: invalid message: %v", err)
        return
    }

    switch {
    case msg.Image != nil:
        h.broadcast(map[string]any{"image": *msg.Image})
    case msg.Join != nil:
        h.handleJoin(c, *msg.Join, msg.Enemy)
    case msg.Battle != nil:
        voteEnabled := false
        if msg.VoteEnabled != nil {
            voteEnabled = *msg.VoteEnabled
        }
        h.handleBattle(*msg.Battle, voteEnabled)
    case msg.Attack != nil:
        h.handleAttack(*msg.Attack)
    case msg.Clean != nil && *msg.Clean:
        h.handleClean()
    case msg.Vote != nil:
        var voter string
        if msg.User != nil {
            voter = *msg.User
        } else {
            for id, cli := range h.tyrantIDToClient {
                if cli == c { voter = id; break }
            }
        }
        h.handleVote(c, voter, *msg.Vote)
    default:
        // ignore
    }
}

func (h *Hub) handleClean() {
    h.mu.Lock()
    // stop battle
    h.inBattle = false
    h.currentActor = ""
    // remove only enemies
    for id, p := range h.participants {
        if p.Enemy {
            delete(h.participants, id)
            delete(h.tyrantIDToClient, id)
        } else {
            // reset ally HP/PP for next battle readiness
            p.CurrentHP = p.FullHP
            for _, v := range p.AttackPP {
                if v != nil {
                    v.Current = v.Full
                }
            }
        }
    }
    h.computeTurnOrderLocked()
    turns := h.turnsViewLocked()
    h.mu.Unlock()

    h.broadcast(map[string]any{"clean": true, "turns": turns})
}

func (h *Hub) handleVote(c *Client, voterID string, choice string) {
    h.mu.Lock()
    if !h.votingActive {
        h.mu.Unlock()
        if c != nil { _ = c.conn.WriteJSON(map[string]any{"error": "voting not active"}) }
        return
    }
    // only allies can vote
    p := h.participants[voterID]
    if p == nil || p.Enemy {
        h.mu.Unlock()
        if c != nil { _ = c.conn.WriteJSON(map[string]any{"error": "only allies can vote"}) }
        return
    }
    if h.votedAllies[voterID] {
        counts := map[string]int{"UNTIL_DEATH": h.voteUntilDeath, "TO_PARTY": h.voteToParty}
        h.mu.Unlock()
        if c != nil { _ = c.conn.WriteJSON(map[string]any{"voting": counts}) }
        return
    }
    switch choice {
    case "UNTIL_DEATH":
        h.voteUntilDeath++
    case "TO_PARTY":
        h.voteToParty++
    default:
        h.mu.Unlock()
        if c != nil { _ = c.conn.WriteJSON(map[string]any{"error": "invalid vote"}) }
        return
    }
    h.votedAllies[voterID] = true
    counts := map[string]int{"UNTIL_DEATH": h.voteUntilDeath, "TO_PARTY": h.voteToParty}
    done := len(h.votedAllies) >= h.totalAllies
    if done {
        h.votingActive = false
        h.inBattle = true
        // tie -> TO_PARTY
        result := "TO_PARTY"
        if h.voteUntilDeath > h.voteToParty { result = "UNTIL_DEATH" }
        _ = result
        h.mu.Unlock()
        h.broadcast(map[string]any{"voting": counts})
        turns := h.turnsViewLocked()
        h.broadcast(map[string]any{"battle": "", "turns": turns})
        return
    }
    h.mu.Unlock()
    h.broadcast(map[string]any{"voting": counts})
}

func (h *Hub) handleJoin(c *Client, tyrantID string, enemy *bool) {
    t, err := h.svc.GetTyrant(tyrantID)
    if err != nil {
        // notify only the sender
        _ = c.conn.WriteJSON(map[string]any{"error": "tyrant not found"})
        return
    }
    en := false
    if enemy != nil {
        en = *enemy
    }
    h.mu.Lock()
    if _, exists := h.participants[t.ID]; !exists {
        p := &Participant{
            Tyrant:    t,
            Enemy:     en,
            FullHP:    t.HP,
            CurrentHP: t.HP,
            Alive:     true,
            AttackPP:  make(map[string]*struct{ Full int; Current int }),
        }
        for _, atk := range t.Attacks {
            p.AttackPP[atk.Name] = &struct{ Full int; Current int }{Full: atk.PP, Current: atk.PP}
        }
        h.participants[t.ID] = p
    }
    h.tyrantIDToClient[t.ID] = c
    // recalculate turn order if already in battle
    if h.inBattle {
        h.computeTurnOrderLocked()
    }
    h.mu.Unlock()

    // acknowledge join
    _ = c.conn.WriteJSON(map[string]any{"joined": t.ID, "enemy": en})
}

func (h *Hub) handleBattle(startWith string, voteEnabled bool) {
    h.mu.Lock()
    h.inBattle = !voteEnabled
    h.votingActive = voteEnabled
    // Reset HP/Alive and PP for a new battle
    for _, p := range h.participants {
        p.CurrentHP = p.FullHP
        p.Alive = p.FullHP > 0
        for _, v := range p.AttackPP {
            if v != nil {
                v.Current = v.Full
            }
        }
    }
    h.computeTurnOrderLocked()
    // align start index to provided tyrant if exists
    h.turnIndex = 0
    for i, id := range h.turnOrder {
        if id == startWith {
            h.turnIndex = i
            break
        }
    }
    next := h.nextAliveLocked()
    h.currentActor = next
    if voteEnabled {
        h.voteUntilDeath = 0
        h.voteToParty = 0
        h.votedAllies = make(map[string]bool)
        h.totalAllies = 0
        for _, p := range h.participants {
            if p != nil && !p.Enemy { h.totalAllies++ }
        }
        h.mu.Unlock()
        h.broadcast(map[string]any{"voting": map[string]int{"UNTIL_DEATH": 0, "TO_PARTY": 0}})
        return
    }
    h.mu.Unlock()
    turns := h.turnsViewLocked()
    h.broadcast(map[string]any{"battle": startWith, "turns": turns})
}

func (h *Hub) computeTurnOrderLocked() {
    order := make([]string, 0, len(h.participants))
    for id := range h.participants {
        order = append(order, id)
    }
    sort.Slice(order, func(i, j int) bool {
        return h.participants[order[i]].Tyrant.Speed > h.participants[order[j]].Tyrant.Speed
    })
    h.turnOrder = order
    if h.turnIndex >= len(h.turnOrder) {
        h.turnIndex = 0
    }
}

func (h *Hub) nextAliveLocked() string {
    if len(h.turnOrder) == 0 {
        return ""
    }
    n := len(h.turnOrder)
    for k := 0; k < n; k++ {
        id := h.turnOrder[h.turnIndex]
        h.turnIndex = (h.turnIndex + 1) % n
        p := h.participants[id]
        if p != nil && p.Alive {
            return id
        }
    }
    return ""
}

func (h *Hub) handleAttack(a attackEvent) {
    h.mu.Lock()
    if !h.inBattle {
        // not in battle
        c := h.tyrantIDToClient[a.User]
        h.mu.Unlock()
        if c != nil {
            _ = c.conn.WriteJSON(map[string]any{"error": "not in battle"})
        }
        return
    }
    attacker := h.participants[a.User]
    target := h.participants[a.Target]
    if attacker == nil || target == nil || !attacker.Alive || !target.Alive {
        c := h.tyrantIDToClient[a.User]
        h.mu.Unlock()
        if c != nil {
            msg := "invalid attacker or target"
            if target == nil {
                msg = "target not found"
            }
            _ = c.conn.WriteJSON(map[string]any{"error": msg})
        }
        return
    }
    // enforce turn: only current actor can act
    if h.currentActor != "" && h.currentActor != a.User {
        c := h.tyrantIDToClient[a.User]
        h.mu.Unlock()
        if c != nil {
            _ = c.conn.WriteJSON(map[string]any{"error": "not your turn", "expected": h.currentActor})
        }
        return
    }
    // Basic validation: attack must exist by name on attacker's tyrant
    var atkDef *models.Attack
    for i := range attacker.Tyrant.Attacks {
        if attacker.Tyrant.Attacks[i].Name == a.Attack {
            atkDef = &attacker.Tyrant.Attacks[i]
            break
        }
    }
    if atkDef == nil {
        c := h.tyrantIDToClient[a.User]
        h.mu.Unlock()
        if c != nil {
            _ = c.conn.WriteJSON(map[string]any{"error": "unknown attack"})
        }
        return
    }
    // Check and consume PP
    pp := attacker.AttackPP[a.Attack]
    if pp == nil || pp.Current <= 0 {
        c := h.tyrantIDToClient[a.User]
        h.mu.Unlock()
        if c != nil {
            _ = c.conn.WriteJSON(map[string]any{"error": "no PP left for attack"})
        }
        return
    }
    pp.Current--
    // Damage calculation
    random := rand.Intn(100) + 1 // 1..100
    atkStat := attacker.Tyrant.Attack
    defStat := target.Tyrant.Defense
    power := atkDef.Power
    damage := (atkStat*(random+(power*10)) - defStat) / 200
    if damage < 1 {
        damage = 1
    }
    if random >= 90 {
        damage = damage * 2
    }
    target.CurrentHP -= damage
    if target.CurrentHP <= 0 {
        target.CurrentHP = 0
        target.Alive = false
    }
    // Build HP+PP update snapshot
    snapshot := map[string]any{}
    for id, p := range h.participants {
        ppMap := map[string]any{}
        for name, v := range p.AttackPP {
            ppMap[name] = map[string]any{"fullPP": v.Full, "currentPP": v.Current}
        }
        snapshot[id] = map[string]any{
            "fullHp":    p.FullHP,
            "currentHp": p.CurrentHP,
            "pp":        ppMap,
        }
    }
    // Determine victory
    allEnemiesDown := true
    allAlliesDown := true
    for _, p := range h.participants {
        if p.Enemy && p.Alive {
            allEnemiesDown = false
        }
        if !p.Enemy && p.Alive {
            allAlliesDown = false
        }
    }
    var status any = snapshot
    if allEnemiesDown {
        status = "WIN"
        h.inBattle = false
        // remove only enemies; keep protagonists for future battles
        for id, p := range h.participants {
            if p.Enemy {
                delete(h.participants, id)
                delete(h.tyrantIDToClient, id)
            }
        }
        h.computeTurnOrderLocked()
    } else if allAlliesDown {
        status = "DEFEAT"
        h.inBattle = false
        // remove only enemies; keep protagonists for future battles
        for id, p := range h.participants {
            if p.Enemy {
                delete(h.participants, id)
                delete(h.tyrantIDToClient, id)
            }
        }
        h.computeTurnOrderLocked()
    }
    // Next turn
    if h.inBattle {
        next := h.nextAliveLocked()
        h.currentActor = next
    } else {
        h.currentActor = ""
    }
    turns := h.turnsViewLocked()
    h.mu.Unlock()

    h.broadcast(map[string]any{"updateState": status, "turns": turns})
}

func (h *Hub) broadcast(v any) {
    h.mu.RLock()
    conns := make([]*websocket.Conn, 0, len(h.clients))
    for c := range h.clients {
        conns = append(conns, c.conn)
    }
    h.mu.RUnlock()
    for _, conn := range conns {
        _ = conn.WriteJSON(v)
    }
}

// turnsViewLocked returns the ordered list of upcoming turns starting from currentActor.
func (h *Hub) turnsViewLocked() []map[string]any {
    result := make([]map[string]any, 0, len(h.turnOrder))
    if len(h.turnOrder) == 0 {
        return result
    }
    startIdx := -1
    if h.currentActor != "" {
        for i, id := range h.turnOrder {
            if id == h.currentActor {
                startIdx = i
                break
            }
        }
    }
    if startIdx == -1 {
        startIdx = h.turnIndex % len(h.turnOrder)
    }
    for offset := 0; offset < len(h.turnOrder); offset++ {
        idx := (startIdx + offset) % len(h.turnOrder)
        id := h.turnOrder[idx]
        p := h.participants[id]
        if p == nil || !p.Alive {
            continue
        }
        result = append(result, map[string]any{
            "id":    id,
            "asset": p.Tyrant.Asset,
            "enemy": p.Enemy,
        })
    }
    return result
}


