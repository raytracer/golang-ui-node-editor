package main

import (
	"github.com/golang-ui/nuklear/nk"
	"math"
)

type node struct {
	ID           int
	name         string
	bounds       nk.Rect
	value        float64
	color        nk.Color
	input_count  int
	output_count int
	next         *node
	prev         *node
}

type node_link struct {
	input_id    int
	input_slot  int
	output_id   int
	output_slot int
	in          nk.Vec2
	out         nk.Vec2
}

type node_linking struct {
	active     bool
	node       *node
	input_id   int
	input_slot int
}

type node_editor struct {
	initialized bool
	node_buf    []*node
	links       []*node_link
	begin       *node
	end         *node
	bounds      nk.Rect
	selected    *node
	show_grid   bool
	scrolling   nk.Vec2
	linking     node_linking
}

func node_editor_push(editor *node_editor, node *node) {
	if editor.begin == nil {
		node.next = nil
		node.prev = nil
		editor.begin = node
		editor.end = node
	} else {
		node.prev = editor.end
		if editor.end != nil {
			editor.end.next = node
		}
		node.next = nil
		editor.end = node
	}
}

func node_editor_pop(editor *node_editor, node *node) {
	if node.next != nil {
		node.next.prev = node.prev
	}
	if node.prev != nil {
		node.prev.next = node.next
	}
	if editor.end == node {
		editor.end = node.prev
	}
	if editor.begin == node {
		editor.begin = node.next
	}
	node.next = nil
	node.prev = nil
}

func node_editor_find(editor *node_editor, ID int) *node {
	iter := editor.begin
	for iter != nil {
		if iter.ID == ID {
			return iter
		}
		iter = iter.next
	}
	return nil
}

var IDs = 0

func node_editor_add(editor *node_editor, name string, bounds nk.Rect,
	col nk.Color, in_count int, out_count int) {
	node := new(node)
	node.ID = IDs
	IDs++
	node.value = 0
	node.color = nk.NkRgb(255, 0, 0)
	node.input_count = in_count
	node.output_count = out_count
	node.color = col
	node.bounds = bounds
	node.name = name
	editor.node_buf = append(editor.node_buf, node)
	node_editor_push(editor, node)
}

func node_editor_link(editor *node_editor, in_id int, in_slot int, out_id int, out_slot int) {
	link := new(node_link)
	editor.links = append(editor.links, link)
	link.input_id = in_id
	link.input_slot = in_slot
	link.output_id = out_id
	link.output_slot = out_slot
}

func node_editor_init(editor *node_editor) {
	node_editor_add(editor, "Source", nk.NkRect(40, 10, 180, 220), nk.NkRgb(255, 0, 0), 0, 1)
	node_editor_add(editor, "Source", nk.NkRect(40, 260, 180, 220), nk.NkRgb(0, 255, 0), 0, 1)
	node_editor_add(editor, "Combine", nk.NkRect(400, 100, 180, 220), nk.NkRgb(0, 0, 255), 2, 2)
	node_editor_link(editor, 0, 0, 2, 0)
	node_editor_link(editor, 1, 0, 2, 1)
	editor.show_grid = true
}

var nodeEditor node_editor

func node_editor_draw(ctx *nk.Context) {
	var total_space nk.Rect
	in := ctx.Input()
	var canvas *nk.CommandBuffer
	var updated *node
	nodedit := &nodeEditor

	if !nodeEditor.initialized {
		node_editor_init(&nodeEditor)
		nodeEditor.initialized = true
	}

	update := nk.NkBegin(ctx, s("NodeEdit"), nk.NkRect(0, 0, 800, 600), nk.WindowBorder|nk.WindowMovable|nk.WindowScalable|nk.WindowMinimizable|nk.WindowTitle)
	if update > 0 {
		/* allocate complete window space */
		canvas = nk.NkWindowGetCanvas(ctx)
		total_space = nk.NkWindowGetContentRegion(ctx)
		nk.NkLayoutSpaceBegin(ctx, nk.Static, total_space.H(), int32(len(nodedit.node_buf)))

		it := nodedit.begin

		/* display grid */
		if nodedit.show_grid {
			size := nk.NkLayoutSpaceBounds(ctx)
			grid_size := 32.0
			grid_color := nk.NkRgb(50, 50, 50)
			for x := math.Mod(float64(size.X()-nodedit.scrolling.X()), grid_size); x < float64(size.W()); x += grid_size {
				nk.NkStrokeLine(canvas, float32(x)+size.X(), size.Y(), float32(x)+size.X(), size.Y()+size.H(), 1.0, grid_color)
			}
			for y := math.Mod(float64(size.Y()-nodedit.scrolling.Y()), grid_size); y < float64(size.H()); y += grid_size {
				nk.NkStrokeLine(canvas, size.X(), float32(y)+size.Y(), size.X()+size.W(), float32(y)+size.Y(), 1.0, grid_color)
			}
		}

		/* execute each node as a movable group */
		var node *nk.Panel
		for it != nil {
			/* calculate scrolled node window position and size */
			nk.NkLayoutSpacePush(ctx, nk.NkRect(it.bounds.X()-nodedit.scrolling.X(),
				it.bounds.Y()-nodedit.scrolling.Y(), it.bounds.W(), it.bounds.H()))

			/* execute node window */
			groupBegin := nk.NkGroupBegin(ctx, s(it.name), nk.WindowMovable|nk.WindowNoScrollbar|nk.WindowBorder|nk.WindowTitle)
			if groupBegin > 0 {
				/* always have last selected node on top */
				node = nk.NkWindowGetPanel(ctx)
				cond1 := nk.NkInputMouseClicked(in, nk.ButtonLeft, *node.Bounds()) > 0
				cond2 := !(it.prev != nil && nk.NkInputMouseClicked(in, nk.ButtonLeft, nk.NkLayoutSpaceRectToScreen(ctx, *node.Bounds())) > 0)
				cond3 := nodedit.end != it
				if cond1 && cond2 && cond3 {
					updated = it
				}

				/* ================= NODE CONTENT =====================*/
				nk.NkLayoutRowDynamic(ctx, 25, 1)
				nk.NkButtonColor(ctx, it.color)
				it.color.SetR(nk.Byte(nk.NkPropertyi(ctx, s("#R:"), 0, int32(it.color.R()), 255, 1, 1)))
				it.color.SetG(nk.Byte(nk.NkPropertyi(ctx, s("#G:"), 0, int32(it.color.G()), 255, 1, 1)))
				it.color.SetB(nk.Byte(nk.NkPropertyi(ctx, s("#B:"), 0, int32(it.color.B()), 255, 1, 1)))
				it.color.SetA(nk.Byte(nk.NkPropertyi(ctx, s("#A:"), 0, int32(it.color.A()), 255, 1, 1)))
				///* ====================================================*/
				nk.NkGroupEnd(ctx)
			}

			/* node connector and linking */
			var bounds nk.Rect
			bounds = nk.NkLayoutSpaceRectToLocal(ctx, *node.Bounds())
			bounds_new := nk.NkRect(bounds.X()+nodedit.scrolling.X(), bounds.Y()+nodedit.scrolling.Y(), bounds.W(), bounds.H())
			it.bounds = bounds_new

			/* output connector */
			output_space := node.Bounds().H() / float32(it.output_count+1)
			for i := 0; i < it.output_count; i++ {
				circle := nk.NkRect(node.Bounds().X()+node.Bounds().W()-4, node.Bounds().Y()+output_space*float32(i+1), 8, 8)
				nk.NkFillCircle(canvas, circle, nk.NkRgb(100, 100, 100))

				/* start linking process */
				if nk.NkInputHasMouseClickDownInRect(in, nk.ButtonLeft, circle, nk.True) > 0 {
					nodedit.linking.active = true
					nodedit.linking.node = it
					nodedit.linking.input_id = it.ID
					nodedit.linking.input_slot = i
				}

				/* draw curve from linked node slot to mouse position */
				if nodedit.linking.active && nodedit.linking.node == it &&
					nodedit.linking.input_slot == i {
					x0, y0 := circle.X()+3, circle.Y()+3
					x1, y1 := in.Mouse().Pos()

					nk.NkStrokeCurve(canvas, x0, y0, x0+50.0, y0,
						float32(x1)-50.0, float32(y1), float32(x1), float32(y1), 1.0, nk.NkRgb(100, 100, 100))
				}
			}

			/* input connector */
			input_space := node.Bounds().H() / float32(it.input_count+1)
			for i := 0; i < it.input_count; i++ {
				circle := nk.NkRect(node.Bounds().X()-4, node.Bounds().Y()+input_space*float32(i+1), 8, 8)
				nk.NkFillCircle(canvas, circle, nk.NkRgb(100, 100, 100))
				if nk.NkInputIsMouseReleased(in, nk.ButtonLeft) > 0 &&
					nk.NkInputIsMouseHoveringRect(in, circle) > 0 &&
					nodedit.linking.active && nodedit.linking.node != it {
					nodedit.linking.active = false
					node_editor_link(nodedit, nodedit.linking.input_id,
						nodedit.linking.input_slot, it.ID, i)
				}
			}

			it = it.next
		}

		/* reset linking connection */
		if nodedit.linking.active && nk.NkInputIsMouseReleased(in, nk.ButtonLeft) > 0 {
			nodedit.linking.active = false
			nodedit.linking.node = nil
		}

		/* draw each link */
		for _, link := range nodedit.links {
			ni := node_editor_find(nodedit, link.input_id)
			no := node_editor_find(nodedit, link.output_id)
			spacei := node.Bounds().H() / float32((ni.output_count)+1)
			spaceo := node.Bounds().H() / float32((no.input_count)+1)
			l0 := nk.NkLayoutSpaceToScreen(ctx,
				nk.NkVec2(ni.bounds.X()+ni.bounds.W(), 3.0+ni.bounds.Y()+spacei*float32(link.input_slot+1)))
			l1 := nk.NkLayoutSpaceToScreen(ctx,
				nk.NkVec2(no.bounds.X(), 3.0+no.bounds.Y()+spaceo*float32(link.output_slot+1)))

			x0 := l0.X() - nodedit.scrolling.X()
			y0 := l0.Y() - nodedit.scrolling.Y()
			x1 := l1.X() - nodedit.scrolling.X()
			y1 := l1.Y() - nodedit.scrolling.Y()
			nk.NkStrokeCurve(canvas, x0, y0, x0+50.0, y0,
				x1-50.0, y1, x1, y1, 1.0, nk.NkRgb(100, 100, 100))
		}

		if updated != nil {
			/* reshuffle nodes to have least recently selected node on top */
			node_editor_pop(nodedit, updated)
			node_editor_push(nodedit, updated)
		}

		/* node selection */
		if nk.NkInputMouseClicked(in, nk.ButtonLeft, nk.NkLayoutSpaceBounds(ctx)) > 0 {
			it = nodedit.begin
			nodedit.selected = nil
			x, y := in.Mouse().Pos()
			nodedit.bounds = nk.NkRect(float32(x), float32(y), 100, 200)
			for it != nil {
				b := nk.NkLayoutSpaceRectToScreen(ctx, it.bounds)
				b_new := nk.NkRect(b.X()-nodedit.scrolling.X(), b.Y()-nodedit.scrolling.Y(), b.W(), b.H())
				if nk.NkInputIsMouseHoveringRect(in, b_new) > 0 {
					nodedit.selected = it
				}
				it = it.next
			}
		}

		/* contextual menu */
		if nk.NkContextualBegin(ctx, 0, nk.NkVec2(100, 220), nk.NkWindowGetBounds(ctx)) > 0 {
			nk.NkLayoutRowDynamic(ctx, 25, 1)
			if nk.NkContextualItemLabel(ctx, s("New"), nk.TextCentered) > 0 {
				node_editor_add(nodedit, "New", nk.NkRect(400, 260, 180, 220),
					nk.NkRgb(255, 255, 255), 1, 2)
			}

			grid_option := "Show Grid"
			if nodedit.show_grid {
				grid_option = "Hide Grid"
			}
			if nk.NkContextualItemLabel(ctx, s(grid_option), nk.TextCentered) > 0 {
				nodedit.show_grid = !nodedit.show_grid
			}
			nk.NkContextualEnd(ctx)
		}

		nk.NkLayoutSpaceEnd(ctx)

		/* window content scrolling */
		if nk.NkInputIsMouseHoveringRect(in, nk.NkWindowGetBounds(ctx)) > 0 &&
			nk.NkInputIsMouseDown(in, nk.ButtonMiddle) > 0 {
			var scroll = nodedit.scrolling
			var deltaX, deltaY = in.Mouse().Delta()
			nodedit.scrolling = nk.NkVec2(scroll.X()+float32(deltaX), scroll.Y()+float32(deltaY))
		}
	}
	nk.NkEnd(ctx)
}
