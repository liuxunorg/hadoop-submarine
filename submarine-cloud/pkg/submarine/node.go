package submarine

import (
	"github.com/apache/submarine/submarine-cloud/pkg/apis/submarine/v1alpha1"
	kapiv1 "k8s.io/api/core/v1"
	"net"
	"sort"
	"time"
)

const (
	// DefaultRedisPort define the default Redis Port
	DefaultRedisPort = "6379"
	// RedisMasterRole redis role master
	redisMasterRole = "master"
	// RedisSlaveRole redis role slave
	redisSlaveRole = "slave"
)

// Node Represent a Redis Node
type Node struct {
	ID              string
	IP              string
	Port            string
	Role            string
	LinkState       string
	MasterReferent  string
	FailStatus      []string
	PingSent        int64
	PongRecv        int64
	ConfigEpoch     int64
	///Slots           []Slot
	///MigratingSlots  map[Slot]string
	///ImportingSlots  map[Slot]string
	ServerStartTime time.Time

	Pod *kapiv1.Pod
}

// NewDefaultNode builds and returns new defaultNode instance
func NewDefaultNode() *Node {
	return &Node{
		Port:           DefaultRedisPort,
		///Slots:          []Slot{},
		///MigratingSlots: map[Slot]string{},
		///ImportingSlots: map[Slot]string{},
	}
}

// Nodes represent a Node slice
type Nodes []*Node

// nodeSorter joins a By function and a slice of Nodes to be sorted.
type nodeSorter struct {
	nodes Nodes
	by    func(p1, p2 *Node) bool // Closure used in the Less method.
}

// Len is part of sort.Interface.
func (s *nodeSorter) Len() int {
	return len(s.nodes)
}

// Swap is part of sort.Interface.
func (s *nodeSorter) Swap(i, j int) {
	s.nodes[i], s.nodes[j] = s.nodes[j], s.nodes[i]
}

// Less is part of sort.Interface. It is implemented by calling the "by" closure in the sorter.
func (s *nodeSorter) Less(i, j int) bool {
	return s.by(s.nodes[i], s.nodes[j])
}

// FindNodeFunc function for finding a Node
// it is use as input for GetNodeByFunc and GetNodesByFunc
type FindNodeFunc func(node *Node) bool

// GetNodesByFunc returns first node found by the FindNodeFunc
func (n Nodes) GetNodesByFunc(f FindNodeFunc) (Nodes, error) {
	nodes := Nodes{}
	for _, node := range n {
		if f(node) {
			nodes = append(nodes, node)
		}
	}
	if len(nodes) == 0 {
		return nodes, nodeNotFoundedError
	}
	return nodes, nil
}

// IsMasterWithSlot anonymous function for searching Master Node withslot
var IsMasterWithSlot = func(n *Node) bool {
	if (n.GetRole() == v1alpha1.SubmarineClusterNodeRoleMaster) && (n.TotalSlots() > 0) {
		return true
	}
	return false
}

// GetRole return the Redis Cluster Node GetRole
func (n *Node) GetRole() v1alpha1.SubmarineClusterNodeRole {
	switch n.Role {
	case redisMasterRole:
		return v1alpha1.SubmarineClusterNodeRoleMaster
	case redisSlaveRole:
		return v1alpha1.SubmarineClusterNodeRoleSlave
	default:
		if n.MasterReferent != "" {
			return v1alpha1.SubmarineClusterNodeRoleSlave
		}
		///if len(n.Slots) > 0 {
		///	return v1alpha1.SubmarineClusterNodeRoleMaster
		///}
	}

	return v1alpha1.SubmarineClusterNodeRoleNone
}

// TotalSlots return the total number of slot
func (n *Node) TotalSlots() int {
	return 1 ///len(n.Slots)
}

// IsSlave anonymous function for searching Slave Node
var IsSlave = func(n *Node) bool {
	return n.GetRole() == v1alpha1.SubmarineClusterNodeRoleSlave
}

// FilterByFunc remove a node from a slice by node ID and returns the slice. If not found, fail silently. Value must be unique
func (n Nodes) FilterByFunc(fn func(*Node) bool) Nodes {
	newSlice := Nodes{}
	for _, node := range n {
		if fn(node) {
			newSlice = append(newSlice, node)
		}
	}
	return newSlice
}

// IsMasterWithNoSlot anonymous function for searching Master Node with no slot
var IsMasterWithNoSlot = func(n *Node) bool {
	if (n.GetRole() == v1alpha1.SubmarineClusterNodeRoleMaster) && (n.TotalSlots() == 0) {
		return true
	}
	return false
}

// By is the type of a "less" function that defines the ordering of its Node arguments.
type by func(p1, p2 *Node) bool

// Sort is a method on the function type, By, that sorts the argument slice according to the function.
func (b by) Sort(nodes Nodes) {
	ps := &nodeSorter{
		nodes: nodes,
		by:    b, // The Sort method's receiver is the function (closure) that defines the sort order.
	}
	sort.Sort(ps)
}

// SortByFunc returns a new ordered NodeSlice, determined by a func defining ‘less’.
func (n Nodes) SortByFunc(less func(*Node, *Node) bool) Nodes {
	result := make(Nodes, len(n))
	copy(result, n)
	by(less).Sort(n)
	return result
}

// IPPort returns join Ip Port string
func (n *Node) IPPort() string {
	return net.JoinHostPort(n.IP, n.Port)
}

// LessByID compare 2 Nodes with there ID
func LessByID(n1, n2 *Node) bool {
	return n1.ID < n2.ID
}
