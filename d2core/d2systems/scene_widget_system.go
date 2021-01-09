package d2systems

import (
	"fmt"
	"image/color"
	"time"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2enum"

	"github.com/gravestench/akara"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2cache"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2geom/rectangle"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2input"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2interface"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2resource"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2util"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2bitmapfont"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2button"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2components"
)

const (
	fontCacheBudget = 64
)

// NewUIWidgetFactory creates a new ui widget factory which is intended
// to be embedded in the game object factory system.
func NewUIWidgetFactory(
	b akara.BaseSystem,
	l *d2util.Logger,
	spriteFactory *SpriteFactory,
	shapeFactory *ShapeSystem,
) *UIWidgetFactory {
	sys := &UIWidgetFactory{
		Logger:            l,
		SpriteFactory:     spriteFactory,
		ShapeSystem:       shapeFactory,
		bitmapFontCache:   d2cache.CreateCache(fontCacheBudget),
		buttonLoadQueue:   make(buttonLoadQueue),
		checkboxLoadQueue: make(checkboxLoadQueue),
		labelLoadQueue:    make(labelLoadQueue),
	}

	sys.BaseSystem = b

	sys.World.AddSystem(sys)

	return sys
}

type buttonLoadQueueEntry struct {
	label, sprite akara.EID
}

type buttonLoadQueue = map[akara.EID]buttonLoadQueueEntry

type checkboxLoadQueueEntry struct {
	sprite akara.EID
}

type checkboxLoadQueue = map[akara.EID]checkboxLoadQueueEntry

type labelLoadQueueEntry struct {
	table, sprite akara.EID
}

type labelLoadQueue = map[akara.EID]labelLoadQueueEntry

// UIWidgetFactory is responsible for creating UI widgets like buttons and tabs
type UIWidgetFactory struct {
	akara.BaseSubscriberSystem
	*d2util.Logger
	*RenderSystem
	*SpriteFactory
	*ShapeSystem
	buttonLoadQueue
	checkboxLoadQueue
	labelLoadQueue
	state              d2enum.SceneState
	bitmapFontCache    d2interface.Cache
	labelsToUpdate     *akara.Subscription
	buttonsToUpdate    *akara.Subscription
	checkboxesToUpdate *akara.Subscription
	Components         struct {
		File           d2components.FileFactory
		Transform      d2components.TransformFactory
		Interactive    d2components.InteractiveFactory
		FontTable      d2components.FontTableFactory
		Palette        d2components.PaletteFactory
		BitmapFont     d2components.BitmapFontFactory
		Label          d2components.LabelFactory
		Button         d2components.ButtonFactory
		Checkbox       d2components.CheckboxFactory
		Sprite         d2components.SpriteFactory
		Color          d2components.ColorFactory
		Texture        d2components.TextureFactory
		Ready          d2components.ReadyFactory
		SceneGraphNode d2components.SceneGraphNodeFactory
	}
}

// Init the ui widget factory, injecting the necessary components
func (t *UIWidgetFactory) Init(world *akara.World) {
	t.World = world

	t.Debug("initializing ui widget factory ...")

	t.setupFactories()
	t.setupSubscriptions()
}

func (t *UIWidgetFactory) setupFactories() {
	t.InjectComponent(&d2components.File{}, &t.Components.File.ComponentFactory)
	t.InjectComponent(&d2components.Transform{}, &t.Components.Transform.ComponentFactory)
	t.InjectComponent(&d2components.Interactive{}, &t.Components.Interactive.ComponentFactory)
	t.InjectComponent(&d2components.FontTable{}, &t.Components.FontTable.ComponentFactory)
	t.InjectComponent(&d2components.Palette{}, &t.Components.Palette.ComponentFactory)
	t.InjectComponent(&d2components.BitmapFont{}, &t.Components.BitmapFont.ComponentFactory)
	t.InjectComponent(&d2components.Label{}, &t.Components.Label.ComponentFactory)
	t.InjectComponent(&d2components.Button{}, &t.Components.Button.ComponentFactory)
	t.InjectComponent(&d2components.Checkbox{}, &t.Components.Checkbox.ComponentFactory)
	t.InjectComponent(&d2components.Sprite{}, &t.Components.Sprite.ComponentFactory)
	t.InjectComponent(&d2components.Color{}, &t.Components.Color.ComponentFactory)
	t.InjectComponent(&d2components.Ready{}, &t.Components.Ready.ComponentFactory)
	t.InjectComponent(&d2components.Texture{}, &t.Components.Texture.ComponentFactory)
	t.InjectComponent(&d2components.SceneGraphNode{}, &t.Components.SceneGraphNode.ComponentFactory)
}

func (t *UIWidgetFactory) setupSubscriptions() {
	labelsToUpdate := t.NewComponentFilter().
		Require(&d2components.Label{}).
		Require(&d2components.Ready{}).
		Build()

	buttonsToUpdate := t.NewComponentFilter().
		Require(&d2components.Button{}).
		Require(&d2components.Ready{}).
		Build()

	checkboxesToUpdate := t.NewComponentFilter().
		Require(&d2components.Checkbox{}).
		Require(&d2components.Ready{}).
		Build()

	t.labelsToUpdate = t.AddSubscription(labelsToUpdate)
	t.buttonsToUpdate = t.AddSubscription(buttonsToUpdate)
	t.checkboxesToUpdate = t.AddSubscription(checkboxesToUpdate)
}

func (t *UIWidgetFactory) boot() {
	if t.RenderSystem == nil {
		return
	}

	if t.RenderSystem.renderer == nil {
		return
	}

	t.state = d2enum.SceneStateBooted
}

// Update processes the load queues and update the widgets. The load queues are necessary because
// UI widgets are composed of a bunch of things, which each need to be loaded by other systems (like the asset loader)
func (t *UIWidgetFactory) Update() {
	start := time.Now()

	if t.state != d2enum.SceneStateBooted {
		t.boot()
		return
	}

	for buttonEID := range t.buttonLoadQueue {
		if time.Since(start) > maxTimePerUpdate {
			return
		}

		t.processButton(buttonEID)
	}

	for checkboxEID := range t.checkboxLoadQueue {
		if time.Since(start) > maxTimePerUpdate {
			return
		}

		t.processCheckbox(checkboxEID)
	}

	for labelEID := range t.labelLoadQueue {
		if time.Since(start) > maxTimePerUpdate {
			return
		}

		t.processLabel(labelEID)
	}

	for _, buttonEID := range t.buttonsToUpdate.GetEntities() {
		if time.Since(start) > maxTimePerUpdate {
			return
		}

		t.updateButton(buttonEID)
	}

	for _, checkboxEID := range t.checkboxesToUpdate.GetEntities() {
		if time.Since(start) > maxTimePerUpdate {
			return
		}

		t.updateCheckbox(checkboxEID)
	}

	for _, labelEID := range t.labelsToUpdate.GetEntities() {
		if time.Since(start) > maxTimePerUpdate {
			return
		}

		t.updateLabel(labelEID)
	}
}

// Label creates a label widget.
//
// The font is assumed to be a path for two files, omitting the file extension
//
// Basically, diablo2 stored bitmap fonts as two files, a glyph table and sprite.
//
// For example, specifying this font: /data/local/FONT/ENG/fontexocet10
//
// will use these two files:
//
// /data/local/FONT/ENG/fontexocet10.dc6
//
// /data/local/FONT/ENG/fontexocet10.tbl
func (t *UIWidgetFactory) Label(text, font, palettePath string) akara.EID {
	tablePath := font + ".tbl"
	spritePath := font + ".dc6"

	labelEID := t.NewEntity()

	tableEID := t.NewEntity()
	t.Components.File.Add(tableEID).Path = tablePath

	spriteEID := t.SpriteFactory.Sprite(0, 0, spritePath, palettePath)

	t.labelLoadQueue[labelEID] = labelLoadQueueEntry{
		table:  tableEID,
		sprite: spriteEID,
	}

	label := t.Components.Label.Add(labelEID)
	label.SetText(text)

	return labelEID
}

func (t *UIWidgetFactory) processLabel(labelEID akara.EID) {
	bmfComponent, found := t.Components.BitmapFont.Get(labelEID)
	if !found {
		t.addBitmapFontForLabel(labelEID)
		return
	}

	bmfComponent.Sprite.BindRenderer(t.renderer)

	label, found := t.Components.Label.Get(labelEID)
	if !found {
		label = t.Components.Label.Add(labelEID)
	}

	label.Font = bmfComponent.BitmapFont

	t.RemoveEntity(t.labelLoadQueue[labelEID].table)

	t.Components.Ready.Add(labelEID)

	delete(t.labelLoadQueue, labelEID)
}

func (t *UIWidgetFactory) updateLabel(labelEID akara.EID) {
	label, found := t.Components.Label.Get(labelEID)
	if !found {
		return
	}

	bmf, found := t.Components.BitmapFont.Get(labelEID)
	if !found {
		return
	}

	if label.Font != bmf.BitmapFont {
		label.Font = bmf.BitmapFont
	}

	col, found := t.Components.Color.Get(labelEID)
	if found {
		label.SetBackgroundColor(col.Color)
	}

	if !label.IsDirty() {
		return
	}

	texture, found := t.RenderSystem.Components.Texture.Get(labelEID)
	if !found {
		texture = t.RenderSystem.Components.Texture.Add(labelEID)
	}

	texture.Texture = t.renderer.NewSurface(label.GetSize())

	label.Render(texture.Texture)
}

func (t *UIWidgetFactory) addBitmapFontForLabel(labelEID akara.EID) {
	// get the load queue
	entry, found := t.labelLoadQueue[labelEID]
	if !found {
		return
	}

	// make sure the components have been loaded (by the asset loader)
	_, tableFound := t.Components.FontTable.Get(entry.table)
	_, spriteFound := t.Components.Sprite.Get(entry.sprite)

	if !(tableFound && spriteFound) {
		return
	}

	// now we check the cache, see if we can just pull a pre-rendered bitmap font
	tableFile, found := t.Components.File.Get(entry.table)
	if !found {
		return
	}

	sprite, found := t.Components.Sprite.Get(entry.sprite)
	if !found {
		return
	}

	cacheKey := fontCacheKey(tableFile.Path, sprite.SpritePath, sprite.PalettePath)

	if iface, found := t.bitmapFontCache.Retrieve(cacheKey); found {
		// we found it, add the bitmap font component and set the embedded struct to what we retrieved
		t.Components.BitmapFont.Add(labelEID).BitmapFont = iface.(*d2bitmapfont.BitmapFont)
		t.Components.Ready.Add(labelEID)

		delete(t.labelLoadQueue, labelEID)

		return
	}

	bmf := t.createBitmapFont(entry)
	if bmf == nil {
		return
	}

	// we need to create and cache the bitmap font
	if err := t.bitmapFontCache.Insert(cacheKey, bmf, 1); err != nil {
		t.Warning(err.Error())
	}

	t.Components.BitmapFont.Add(labelEID).BitmapFont = bmf
}

func (t *UIWidgetFactory) createBitmapFont(entry labelLoadQueueEntry) *d2bitmapfont.BitmapFont {
	// make sure the components have been loaded (by the asset loader)
	table, tableFound := t.Components.FontTable.Get(entry.table)
	sprite, spriteFound := t.Components.Sprite.Get(entry.sprite)

	if !(tableFound && spriteFound) || sprite.Sprite == nil || table.Data == nil {
		return nil
	}

	return d2bitmapfont.New(sprite.Sprite, table.Data, color.White)
}

func fontCacheKey(t, s, p string) string {
	return fmt.Sprintf("%s::%s::%s", t, s, p)
}

const isNotSegmented = -1

// Button creates a button ui widget
func (t *UIWidgetFactory) Button(x, y float64, btnType d2button.ButtonType, text string) akara.EID {
	buttonEID := t.NewEntity()

	button := t.Components.Button.Add(buttonEID)

	btnTrs := t.Components.Transform.Add(buttonEID)
	btnTrs.Translation.X, btnTrs.Translation.Y = x, y

	layout := d2button.GetLayout(btnType)

	button.Layout = layout

	var labelEID akara.EID

	if layout.FontPath != "" && layout.PalettePath != "" {
		labelEID = t.Label(text, layout.FontPath, d2resource.PaletteUnits)
	} else {
		// we just make a temporary label and mark it as ready, nothing to do
		labelEID = t.NewEntity()
		t.Components.Ready.Add(labelEID)
	}

	img, pal := layout.SpritePath, layout.PalettePath
	sx, sy, base := layout.XSegments, layout.YSegments, layout.BaseFrame

	spriteEID := t.SegmentedSprite(0, 0, img, pal, sx, sy, base)

	entry := buttonLoadQueueEntry{
		label:  labelEID,
		sprite: spriteEID,
	}

	t.buttonLoadQueue[buttonEID] = entry

	return buttonEID
}

func (t *UIWidgetFactory) processButton(buttonEID akara.EID) {
	// get the queue entry
	entry, found := t.buttonLoadQueue[buttonEID]
	if !found {
		return
	}

	labelEID, spriteEID := entry.label, entry.sprite

	// check if the label and sprite are ready to be used
	_, labelReady := t.Components.Ready.Get(labelEID)
	_, spriteReady := t.Components.Ready.Get(spriteEID)

	if !(labelReady && spriteReady) {
		return
	}

	button, found := t.Components.Button.Get(buttonEID)
	if !found {
		button = t.Components.Button.Add(buttonEID)
	}

	buttonNode := t.Components.SceneGraphNode.Add(buttonEID)

	sprite, found := t.Components.Sprite.Get(spriteEID)
	if found {
		button.Sprite = sprite.Sprite

		t.Components.SceneGraphNode.Add(spriteEID).SetParent(buttonNode.Node)
	}

	_, found = t.Components.Label.Get(labelEID)
	if found {
		t.Components.SceneGraphNode.Add(labelEID).SetParent(buttonNode.Node)
	}

	t.processButtonStates(buttonEID)
}

func (t *UIWidgetFactory) processButtonStates(buttonEID akara.EID) {
	button, _ := t.Components.Button.Get(buttonEID)
	baseFrame := button.Layout.BaseFrame

	hasBeenProcessed := button.States.Normal > 0

	if hasBeenProcessed {
		t.renderButtonStates(buttonEID)
		return
	}

	img, pal := button.Layout.SpritePath, button.Layout.PalettePath
	sx, sy := button.Layout.XSegments, button.Layout.YSegments

	// by default, all other states are whatever the normal state is
	normal := t.SegmentedSprite(0, 0, img, pal, sx, sy, baseFrame)
	button.States.Normal = normal
	button.States.Pressed = normal
	button.States.Toggled = normal
	button.States.PressedToggled = normal
	button.States.Disabled = normal

	// if it's got other states (most buttons do...), then we handle it
	if button.Layout.HasImage && button.Layout.AllowFrameChange {
		button.States.Pressed = t.SegmentedSprite(0, 0, img, pal, sx, sy, baseFrame+d2button.ButtonStatePressed)
		button.States.Toggled = t.SegmentedSprite(0, 0, img, pal, sx, sy, baseFrame+d2button.ButtonStateToggled)
		button.States.PressedToggled = t.SegmentedSprite(0, 0, img, pal, sx, sy, baseFrame+d2button.ButtonStatePressedToggled)

		// also, not all buttons have a disabled state
		// this stupid fucking -1 needs to be a constant
		if button.Layout.DisabledFrame != isNotSegmented {
			button.States.Disabled = t.SegmentedSprite(0, 0, img, pal, sx, sy, button.Layout.DisabledFrame)
		}
	}
}

func (t *UIWidgetFactory) renderButtonStates(buttonEID akara.EID) {
	button, _ := t.Components.Button.Get(buttonEID)

	_, rdyNormal := t.Components.Ready.Get(button.States.Normal)
	_, rdyPressed := t.Components.Ready.Get(button.States.Pressed)
	_, rdyToggled := t.Components.Ready.Get(button.States.Toggled)
	_, rdyPressedToggled := t.Components.Ready.Get(button.States.PressedToggled)
	_, rdyDisabled := t.Components.Ready.Get(button.States.Disabled)

	ready := rdyNormal && rdyPressed && rdyToggled && rdyPressedToggled && rdyDisabled

	if !ready {
		return
	}

	t.Components.Ready.Add(buttonEID)

	delete(t.buttonLoadQueue, buttonEID)
}

func (t *UIWidgetFactory) updateButton(buttonEID akara.EID) {
	button, btnFound := t.Components.Button.Get(buttonEID)

	if !btnFound {
		return
	}

	normal, f1 := t.Components.Texture.Get(button.States.Normal)
	pressed, f2 := t.Components.Texture.Get(button.States.Pressed)
	toggled, f3 := t.Components.Texture.Get(button.States.Toggled)
	pressedToggled, f4 := t.Components.Texture.Get(button.States.PressedToggled)
	disabled, f5 := t.Components.Texture.Get(button.States.Disabled)

	if !(f1 && f2 && f3 && f4 && f5) {
		return
	}

	button.Surfaces.Normal = normal.Texture
	button.Surfaces.Pressed = pressed.Texture
	button.Surfaces.Toggled = toggled.Texture
	button.Surfaces.PressedToggled = pressedToggled.Texture
	button.Surfaces.Disabled = disabled.Texture

	texture, found := t.Components.Texture.Get(buttonEID)
	if !found {
		texture = t.Components.Texture.Add(buttonEID)
	}

	texture.Texture = button.GetCurrentTexture()
}

// Checkbox creates a checkbox ui widget. A Checkbox widget is composed of a Checkbox component that tracks the logic,
// and a SegmentedSprite to be displayed in the scene.
func (t *UIWidgetFactory) Checkbox(x, y float64, checkedState, enabled bool, callback func(akara.Component) bool) akara.EID {
	checkboxEID := t.NewEntity()

	checkbox := t.Components.Checkbox.Add(checkboxEID)
	checkbox.Layout.X, checkbox.Layout.Y = x, y
	checkbox.Layout.ClickableRect = rectangle.New(x, y, float64(checkbox.Layout.FixedWidth), float64(checkbox.Layout.FixedHeight))

	checkboxTrs := t.Components.Transform.Add(checkboxEID)
	checkboxTrs.Translation.X, checkboxTrs.Translation.Y = x, y

	checkbox.Checkbox.SetPressed(checkedState)
	checkbox.Checkbox.SetEnabled(enabled)
	checkbox.Checkbox.OnActivated(callback)

	img, pal := checkbox.Layout.SpritePath, checkbox.Layout.PalettePath
	sx, sy, base := checkbox.Layout.XSegments, checkbox.Layout.YSegments, checkbox.Layout.BaseFrame

	spriteEID := t.SpriteFactory.SegmentedSprite(x, y, img, pal, sx, sy, base)

	entry := checkboxLoadQueueEntry{
		sprite: spriteEID,
	}

	t.checkboxLoadQueue[checkboxEID] = entry

	return checkboxEID
}

// processCheckbox creates a checkbox after all of the prerequisite components are ready.
// This adds interactivity and prepares the checkbox for rendering in the scene.
func (t *UIWidgetFactory) processCheckbox(checkboxEID akara.EID) {
	// get the queue entry
	entry, found := t.checkboxLoadQueue[checkboxEID]
	if !found {
		return
	}

	spriteEID := entry.sprite

	// check if sprite is ready to be used
	if _, spriteReady := t.Components.Ready.Get(spriteEID); !spriteReady {
		return
	}

	checkbox, found := t.Components.Checkbox.Get(checkboxEID)
	if !found {
		checkbox = t.Components.Checkbox.Add(checkboxEID)
	}

	checkboxNode := t.Components.SceneGraphNode.Add(checkboxEID)

	sprite, found := t.Components.Sprite.Get(spriteEID)
	if found {
		checkbox.Sprite = sprite.Sprite

		t.Components.SceneGraphNode.Add(spriteEID).SetParent(checkboxNode.Node)
	}

	interactive := t.Components.Interactive.Add(checkboxEID)
	interactive.InputVector.SetMouseButton(d2input.MouseButtonLeft)
	interactive.CursorPosition = checkbox.Layout.ClickableRect
	interactive.Callback = func() bool {
		return checkbox.Activate(checkbox)
	}

	t.Components.Texture.Add(checkboxEID)

	t.Components.Ready.Add(checkboxEID)

	delete(t.checkboxLoadQueue, checkboxEID)
}

// updateCheckbox refreshes the rendering logic for a checkbox,
// causing any changes that have occurred to appear in the scene.
func (t *UIWidgetFactory) updateCheckbox(checkboxEID akara.EID) {
	checkbox, found := t.Components.Checkbox.Get(checkboxEID)
	if !found {
		return
	}

	checkbox.Update()

	checkboxTexture, _ := t.Components.Texture.Get(checkboxEID)
	checkboxTexture.Texture = checkbox.Sprite.GetCurrentFrameSurface()
}
