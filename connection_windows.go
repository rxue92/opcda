//go:build windows
// +build windows

package opcda

import (
	"errors"
	"fmt"
	"sync"
	"time"

	ole "github.com/go-ole/go-ole"
	"github.com/go-ole/go-ole/oleutil"
)

func init() {
	OleInit()
}

// OleInit initializes OLE.
func OleInit() {
	ole.CoInitializeEx(0, 0)
}

// OleRelease realeses OLE resources in opcAutomation.
func OleRelease() {
	ole.CoUninitialize()
}

// AutomationObject loads the OPC Automation Wrapper and handles to connection to the OPC Server.
type AutomationObject struct {
	unknown *ole.IUnknown
	opc     *ole.IDispatch
}

// CreateBrowser returns the OPCBrowser object from the OPCServer.
// It only works if there is a successful connection.
func (ao *AutomationObject) CreateBrowser() (*Tree, error) {
	// check if server is running, if not return error
	if !ao.IsConnected() {
		return nil, errors.New("cannot create browser because we are not connected")
	}

	// create browser
	browser, err := oleutil.CallMethod(ao.opc, "CreateBrowser")
	if err != nil {
		return nil, errors.New("failed to create OPCBrowser")
	}

	// move to root
	oleutil.MustCallMethod(browser.ToIDispatch(), "MoveToRoot")

	// create tree
	root := Tree{"root", nil, []*Tree{}, []Leaf{}}
	buildTree(browser.ToIDispatch(), &root)

	return &root, nil
}

// buildTree runs through the OPCBrowser and creates a tree with the OPC tags
func buildTree(browser *ole.IDispatch, branch *Tree) {
	var count int32

	logger.Println("Entering branch:", branch.Name)

	// loop through leafs
	oleutil.MustCallMethod(browser, "ShowLeafs").ToIDispatch()
	count = oleutil.MustGetProperty(browser, "Count").Value().(int32)

	logger.Println("\tLeafs count:", count)

	for i := 1; i <= int(count); i++ {

		item := oleutil.MustCallMethod(browser, "Item", i).Value()
		tag := oleutil.MustCallMethod(browser, "GetItemID", item).Value()

		l := Leaf{Name: item.(string), ItemId: tag.(string)}

		logger.Println("\t", i, l)

		branch.Leaves = append(branch.Leaves, l)
	}

	// loop through branches
	oleutil.MustCallMethod(browser, "ShowBranches").ToIDispatch()
	count = oleutil.MustGetProperty(browser, "Count").Value().(int32)

	logger.Println("\tBranches count:", count)

	for i := 1; i <= int(count); i++ {

		nextName := oleutil.MustCallMethod(browser, "Item", i).Value()

		logger.Println("\t", i, "next branch:", nextName)

		// move down
		oleutil.MustCallMethod(browser, "MoveDown", nextName)

		// recursively populate tree
		nextBranch := Tree{nextName.(string), branch, []*Tree{}, []Leaf{}}
		branch.Branches = append(branch.Branches, &nextBranch)
		buildTree(browser, &nextBranch)

		// move up and set branches again
		oleutil.MustCallMethod(browser, "MoveUp")
		oleutil.MustCallMethod(browser, "ShowBranches").ToIDispatch()
	}

	logger.Println("Exiting branch:", branch.Name)

}

// Connect establishes a connection to the OPC Server on node.
// It returns a reference to AutomationItems and error message.
func (ao *AutomationObject) Connect(server string, node string) (*AutomationItems, error) {

	// make sure there is not active connection before trying to connect
	ao.disconnect()

	// try to connect to opc server and check for error
	logger.Printf("Connecting to %s on node %s\n", server, node)
	_, err := oleutil.CallMethod(ao.opc, "Connect", server, node)
	if err != nil {
		logger.Println("Connection failed:", err)
		return nil, errors.New("connection failed: " + refineOleError(err).Error())
	}

	// set up opc groups and items
	opcGroups, err := oleutil.GetProperty(ao.opc, "OPCGroups")
	if err != nil {
		//logger.Println(err)
		return nil, errors.New("cannot get OPCGroups property")
	}
	opcGrp, err := oleutil.CallMethod(opcGroups.ToIDispatch(), "Add")
	if err != nil {
		// logger.Println(err)
		return nil, errors.New("cannot add new OPC Group")
	}
	addItemObject, err := oleutil.GetProperty(opcGrp.ToIDispatch(), "OPCItems")
	if err != nil {
		// logger.Println(err)
		return nil, errors.New("cannot get OPC Items")
	}

	opcGroups.ToIDispatch().Release()
	opcGrp.ToIDispatch().Release()

	logger.Println("Connected.")

	return NewAutomationItems(addItemObject.ToIDispatch()), nil
}

// TryConnect loops over the nodes array and tries to connect to any of the servers.
func (ao *AutomationObject) TryConnect(server string, nodes []string) (*AutomationItems, error) {
	var errResult string
	for _, node := range nodes {
		items, err := ao.Connect(server, node)
		if err == nil {
			return items, err
		}
		errResult = errResult + err.Error()
	}
	return nil, errors.New("TryConnect was not successful: " + errResult)
}

// IsConnected check if the server is properly connected and up and running.
func (ao *AutomationObject) IsConnected() bool {
	if ao.opc == nil {
		return false
	}
	stateVt, err := oleutil.GetProperty(ao.opc, "ServerState")
	if err != nil {
		logger.Println("GetProperty call for ServerState failed", err)
		return false
	}
	if stateVt.Value().(int32) != OPCRunning {
		return false
	}
	return true
}

// GetOPCServers returns a list of Prog ID on the specified node
func (ao *AutomationObject) GetOPCServers(node string) []string {
	progids, err := oleutil.CallMethod(ao.opc, "GetOPCServers", node)
	if err != nil {
		logger.Println("GetOPCServers call failed.")
		return []string{}
	}

	var servers_found []string
	for _, v := range progids.ToArray().ToStringArray() {
		if v != "" {
			servers_found = append(servers_found, v)
		}
	}
	return servers_found
}

func (ao *AutomationObject) PublicGroupNames() []string {
	if !ao.IsConnected() {
		return []string{}
	}
	publicGroups, err := oleutil.GetProperty(ao.opc, "PublicGroupNames")
	if err != nil {
		logger.Println("GetProperty call for PublicGroupNames failed", err)
		return []string{}
	}
	var publicGroups_found []string
	for _, v := range publicGroups.ToArray().ToStringArray() {
		if v != "" {
			publicGroups_found = append(publicGroups_found, v)
		}
	}
	return publicGroups_found
}

// Disconnect checks if connected to server and if so, it calls 'disconnect'
func (ao *AutomationObject) disconnect() {
	if ao.IsConnected() {
		_, err := oleutil.CallMethod(ao.opc, "Disconnect")
		if err != nil {
			logger.Println("Failed to disconnect.")
		}
	}
}

// Close releases the OLE objects in the AutomationObject.
func (ao *AutomationObject) Close() {
	if ao.opc != nil {
		ao.disconnect()
		ao.opc.Release()
	}
	if ao.unknown != nil {
		ao.unknown.Release()
	}
}

// NewAutomationObject connects to the COM object based on available wrappers.
func NewAutomationObject() (*AutomationObject, error) {
	wrappers := []string{"OPC.Automation.1", "Graybox.OPC.DAWrapper.1"}
	var err error
	var unknown *ole.IUnknown
	for _, wrapper := range wrappers {
		unknown, err = oleutil.CreateObject(wrapper)
		if err == nil {
			logger.Println("Loaded OPC Automation object with wrapper", wrapper)
			break
		}
		// 使用OPC.Automation.1 需要将GOARCH环境变量改为386，否则报没有注册类
		// powershell: $env:GOARCH=386
		logger.Printf("Could not load OPC Automation object with wrapper [%s], err=[%v]\n", wrapper, err)
	}
	if err != nil {
		return nil, err
	}

	opc, err := unknown.QueryInterface(ole.IID_IDispatch)
	if err != nil {
		fmt.Println("Could not QueryInterface:", err)
		return nil, err
	}
	object := AutomationObject{
		unknown: unknown,
		opc:     opc,
	}
	return &object, nil
}

// AutomationItems store the OPCItems from OPCGroup and does the bookkeeping
// for the individual OPC items. Tags can added, removed, and read.
type AutomationItems struct {
	addItemObject *ole.IDispatch
	items         map[string]*itemWrap
}

type itemWrap struct {
	*ole.IDispatch
	writeOnly bool // if true, conn.Read() will not read this item
}

// addSingle adds the tag and returns an error. Client handles are not implemented yet.
func (ai *AutomationItems) addSingle(tag string) error {
	clientHandle := int32(1)
	item, err := oleutil.CallMethod(ai.addItemObject, "AddItem", tag, clientHandle)
	if err != nil {
		return errors.New(tag + ":" + err.Error())
	}
	// if item does not belong to address space, item.Val is nil
	if item.Val == 0 {
		return errors.New(tag + ": val is 0")
	}
	disp := item.ToIDispatch()
	if disp == nil {
		return errors.New(tag + ": could not get IDispatch")
	}
	ai.items[tag] = &itemWrap{disp, false}
	return nil
}

// Add accepts a variadic parameters of tags.
func (ai *AutomationItems) Add(tags ...string) error {
	var errResult string
	for _, tag := range tags {
		err := ai.addSingle(tag)
		if err != nil {
			errResult = err.Error() + ";;" + errResult
		}
	}
	if errResult == "" {
		return nil
	}
	return errors.New(errResult)
}

// Remove removes the tag.
func (ai *AutomationItems) Remove(tag string) {
	item, ok := ai.items[tag]
	if ok {
		item.Release()
	}
	delete(ai.items, tag)
}

/*
 * FIX:
 * some opc servers sometimes returns an int32 Quality, that produces panic
 */
func ensureInt16(q interface{}) int16 {
	if v16, ok := q.(int16); ok {
		return v16
	}
	if v32, ok := q.(int32); ok && v32 >= -32768 && v32 < 32768 {
		return int16(v32)
	}
	return 0
}

// readFromOPC reads from the server and returns an Item and error.
func (ai *AutomationItems) readFromOpc(opcitem *ole.IDispatch) (Item, error) {
	v := ole.NewVariant(ole.VT_R4, 0)
	q := ole.NewVariant(ole.VT_INT, 0)
	ts := ole.NewVariant(ole.VT_DATE, 0)

	//read tag from opc server and monitor duration in seconds
	_, err := oleutil.CallMethod(opcitem, "Read", OPCCache, &v, &q, &ts)

	if err != nil {
		return Item{}, err
	}

	return Item{
		Value:     v.Value(),
		Quality:   ensureInt16(q.Value()), // FIX: ensure the quality value is int16
		Timestamp: ts.Value().(time.Time),
	}, nil
}

// writeToOPC writes value to opc tag and return an error
func (ai *AutomationItems) writeToOpc(opcitem *ole.IDispatch, value interface{}) error {
	_, err := oleutil.CallMethod(opcitem, "Write", value)
	if err != nil {
		// TODO: Prometheus Monitoring
		//opcWritesCounter.WithLabelValues("failed").Inc()
		return err
	}
	//opcWritesCounter.WithLabelValues("failed").Inc()
	return nil
}

// Close closes the OLE objects in AutomationItems.
func (ai *AutomationItems) Close() {
	if ai != nil {
		for key, opcitem := range ai.items {
			opcitem.Release()
			delete(ai.items, key)
		}
		ai.addItemObject.Release()
	}
}

// NewAutomationItems returns a new AutomationItems instance.
func NewAutomationItems(opcitems *ole.IDispatch) *AutomationItems {
	ai := AutomationItems{addItemObject: opcitems, items: make(map[string]*itemWrap)}
	return &ai
}

// opcRealServer implements the Connection interface.
// It has the AutomationObject embedded for connecting to the server
// and an AutomationItems to facilitate the OPC items bookkeeping.
type opcConnectionImpl struct {
	*AutomationObject
	*AutomationItems
	Server string
	Nodes  []string
	mu     sync.Mutex
}

// ReadItem returns an Item for a specific tag.
func (conn *opcConnectionImpl) ReadItem(tag string) Item {
	conn.mu.Lock()
	defer conn.mu.Unlock()
	opcitem, ok := conn.AutomationItems.items[tag]
	if ok {
		item, err := conn.AutomationItems.readFromOpc(opcitem.IDispatch)
		if err == nil {
			return item
		}
		logger.Printf("Cannot read %s: %s. Trying to fix.", tag, err)
		conn.fix()
	} else {
		logger.Printf("Tag %s not found. Add it first before reading it.", tag)
	}
	return Item{}
}

// Write writes a value to the OPC Server.
// If tag not found, try add it first.
func (conn *opcConnectionImpl) Write(tag string, value interface{}) error {
	conn.mu.Lock()
	defer conn.mu.Unlock()
	_, ok := conn.AutomationItems.items[tag]
	if !ok {
		err := conn.AutomationItems.addSingle(tag)
		if err != nil {
			return fmt.Errorf("failed to add tag %s: %s", tag, err)
		}
		conn.AutomationItems.items[tag].writeOnly = true
	}
	opcitem := conn.AutomationItems.items[tag]
	return conn.AutomationItems.writeToOpc(opcitem.IDispatch, value)
}

// Read returns a map of the values of all added tags.
func (conn *opcConnectionImpl) Read() map[string]Item {
	conn.mu.Lock()
	defer conn.mu.Unlock()
	allTags := make(map[string]Item)
	for tag, opcitem := range conn.AutomationItems.items {
		if opcitem.writeOnly {
			continue
		}
		item, err := conn.AutomationItems.readFromOpc(opcitem.IDispatch)
		if err != nil {
			logger.Printf("Cannot read %s: %s. Trying to fix.", tag, err)
			conn.fix()
			break
			// logger.Printf("Cannot read %s: %s.", tag, err)
			// continue
		}
		allTags[tag] = item
	}
	return allTags
}

// Tags returns the currently active tags
func (conn *opcConnectionImpl) Tags() []string {
	var tags []string
	if conn.AutomationItems != nil {
		for tag := range conn.AutomationItems.items {
			tags = append(tags, tag)
		}
	}
	return tags
}

// Avoid read during adding or removing items
func (conn *opcConnectionImpl) Add(items ...string) error {
	conn.mu.Lock()
	defer conn.mu.Unlock()
	return conn.AutomationItems.Add(items...)
}

func (conn *opcConnectionImpl) Remove(item string) {
	conn.mu.Lock()
	defer conn.mu.Unlock()
	conn.AutomationItems.Remove(item)
}

// fix tries to reconnect if connection is lost by creating a new connection
// with AutomationObject and creating a new AutomationItems instance.
func (conn *opcConnectionImpl) fix() {
	var err error
	if !conn.IsConnected() {
		for {
			tags := conn.Tags()
			conn.AutomationItems.Close()
			conn.AutomationItems, err = conn.TryConnect(conn.Server, conn.Nodes)
			if err != nil {
				logger.Println(err)
				time.Sleep(100 * time.Millisecond)
				continue
			}
			if conn.Add(tags...) == nil {
				logger.Printf("Added %d tags", len(tags))
			}
			break
		}
	}
}

// Close closes the embedded types.
func (conn *opcConnectionImpl) Close() {
	conn.mu.Lock()
	defer conn.mu.Unlock()
	if conn.AutomationObject != nil {
		conn.AutomationObject.Close()
	}
	if conn.AutomationItems != nil {
		conn.AutomationItems.Close()
	}
}

func (conn *opcConnectionImpl) IsConnected() bool {
	if conn.AutomationObject != nil {
		return conn.AutomationObject.IsConnected()
	}
	return false
}

// NewConnection establishes a connection to the OpcServer object.
func NewConnection(server string, nodes []string, tags []string) (Connection, error) {
	object, err := NewAutomationObject()
	if err != nil {
		return nil, err
	}
	items, err := object.TryConnect(server, nodes)
	if err != nil {
		object.disconnect()
		return &opcConnectionImpl{}, err
	}
	err = items.Add(tags...)
	if err != nil {
		items.Close()
		object.disconnect()
		return &opcConnectionImpl{}, err
	}
	conn := opcConnectionImpl{
		AutomationObject: object,
		AutomationItems:  items,
		Server:           server,
		Nodes:            nodes,
	}

	return &conn, nil
}

// CreateBrowser creates an opc browser representation
func CreateBrowser(server string, nodes []string) (*Tree, error) {
	object, err := NewAutomationObject()
	if err != nil {
		return nil, err
	}
	defer object.Close()
	_, err = object.TryConnect(server, nodes)
	if err != nil {
		return nil, err
	}
	return object.CreateBrowser()
}

type browserImpl struct {
	*AutomationObject
	Server   string
	Nodes    []string
	mu       sync.Mutex
	position string
	browser  *ole.VARIANT
}

func NewBrowser(server string, nodes []string) (Browser, error) {
	object, err := NewAutomationObject()
	if err != nil {
		return nil, err
	}
	items, err := object.TryConnect(server, nodes)
	if err != nil {
		return nil, err
	}
	items.Close()
	// create browser
	browser, err := oleutil.CallMethod(object.opc, "CreateBrowser")
	if err != nil {
		return nil, errors.New("failed to create OPCBrowser")
	}

	// move to root
	oleutil.MustCallMethod(browser.ToIDispatch(), "MoveToRoot")
	return &browserImpl{AutomationObject: object, Server: server, Nodes: nodes, browser: browser}, nil
}

func (b *browserImpl) Close() {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.IsConnected() {
		b.AutomationObject.Close()
	}
	b.browser.ToIDispatch().Release()
}

func (b *browserImpl) MoveTo(branches ...string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.IsConnected() {
		oleutil.MustCallMethod(b.browser.ToIDispatch(), "MoveTo", branches)
	}
}

func (b *browserImpl) MoveToRoot() {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.IsConnected() {
		oleutil.MustCallMethod(b.browser.ToIDispatch(), "MoveToRoot")
	}
}

func (b *browserImpl) MoveUp() {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.IsConnected() {
		oleutil.MustCallMethod(b.browser.ToIDispatch(), "MoveUp")
	}
}

func (b *browserImpl) MoveDown(branch string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.IsConnected() {
		oleutil.MustCallMethod(b.browser.ToIDispatch(), "MoveDown", branch)
	}
}

func (b *browserImpl) Position() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.IsConnected() {
		p := oleutil.MustGetProperty(b.browser.ToIDispatch(), "CurrentPosition").Value().(string)
		b.position = p
	}
	return b.position
}

func (b *browserImpl) ShowBranches() []string {
	b.mu.Lock()
	defer b.mu.Unlock()
	if !b.IsConnected() {
		return []string{}
	}

	branches := make([]string, 0)
	browser := b.browser.ToIDispatch()

	// loop through branches
	oleutil.MustCallMethod(browser, "ShowBranches").ToIDispatch()
	count := oleutil.MustGetProperty(browser, "Count").Value().(int32)

	// logger.Println("\tBranches count:", count)
	for i := 1; i <= int(count); i++ {

		nextName := oleutil.MustCallMethod(browser, "Item", i).Value()

		// logger.Println("\t", i, "next branch:", nextName)
		branches = append(branches, nextName.(string))
	}
	return branches
}

func (b *browserImpl) ShowLeafs() []Leaf {
	b.mu.Lock()
	defer b.mu.Unlock()
	if !b.IsConnected() {
		return []Leaf{}
	}

	leaves := make([]Leaf, 0)
	browser := b.browser.ToIDispatch()
	// loop through leafs
	oleutil.MustCallMethod(browser, "ShowLeafs").ToIDispatch()
	count := oleutil.MustGetProperty(browser, "Count").Value().(int32)

	// logger.Println("\tLeafs count:", count)

	for i := 1; i <= int(count); i++ {

		item := oleutil.MustCallMethod(browser, "Item", i).Value()
		tag := oleutil.MustCallMethod(browser, "GetItemID", item).Value()
		l := Leaf{Name: item.(string), ItemId: tag.(string)}

		// logger.Println("\t", i, l)

		leaves = append(leaves, l)
	}
	return leaves
}
