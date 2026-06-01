//go:build ignore
// +build ignore

// go-mod-graph-local is a standalone web server that visualizes
// go-cqrs-lite internal module dependencies by reading go.mod files
// directly from the local workspace (no external proxy needed).
//
// Usage:
//   go run scripts/go-mod-graph-local/main.go
//   open http://localhost:8765
//
// The server parses all go.mod files in the workspace, extracts
// internal dependencies (github.com/larsartmann/go-cqrs-lite/*),
// and serves an interactive vis.js graph.

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/mod/modfile"
)

type Node struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Group string `json:"group"`
	Level int    `json:"level"`
	Title string `json:"title"`
}

type Edge struct {
	From   string `json:"from"`
	To     string `json:"to"`
	Arrows string `json:"arrows"`
}

type GraphData struct {
	Nodes []Node `json:"nodes"`
	Edges []Edge `json:"edges"`
}

func main() {
	root := "."
	if len(os.Args) > 1 {
		root = os.Args[1]
	}

	graph, err := buildGraph(root)
	if err != nil {
		log.Fatalf("Error building graph: %v", err)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8765"
	}

	http.HandleFunc("/api/graph", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		json.NewEncoder(w).Encode(graph)
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, htmlPage)
	})

	addr := ":" + port
	log.Printf("go-mod-graph-local starting on http://localhost%s", addr)
	log.Println("Press Ctrl+C to stop")
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

func buildGraph(root string) (*GraphData, error) {
	modules := make(map[string]bool)
	edgesMap := make(map[string]bool)
	var edgeList []Edge

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("walk %s: %w", root, err)
		}
		if info.IsDir() {
			name := info.Name()
			if name == "vendor" || name == ".git" || name == "node_modules" || name == ".direnv" ||
				name == "example" {
				return filepath.SkipDir
			}
			return nil
		}
		if info.Name() != "go.mod" {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read %s: %w", path, err)
		}

		mf, err := modfile.Parse(path, data, nil)
		if err != nil {
			return fmt.Errorf("parse %s: %w", path, err)
		}

		moduleName := mf.Module.Mod.Path
		if !strings.HasPrefix(moduleName, "github.com/larsartmann/go-cqrs-lite") {
			return nil
		}

		modules[moduleName] = true

		for _, req := range mf.Require {
			dep := req.Mod.Path
			if !strings.HasPrefix(dep, "github.com/larsartmann/go-cqrs-lite") {
				continue
			}
			if dep == moduleName {
				continue
			}
			edgeKey := moduleName + "->" + dep
			if !edgesMap[edgeKey] {
				edgesMap[edgeKey] = true
				edgeList = append(edgeList, Edge{
					From:   moduleName,
					To:     dep,
					Arrows: "to",
				})
			}
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("build graph from %s: %w", root, err)
	}

	for _, e := range edgeList {
		modules[e.To] = true
	}

	// Compute levels via topological sort (longest path from root)
	levelMap := make(map[string]int)
	for mod := range modules {
		levelMap[mod] = 0
	}

	changed := true
	for changed {
		changed = false
		for _, e := range edgeList {
			newLevel := levelMap[e.To] + 1
			if newLevel > levelMap[e.From] {
				levelMap[e.From] = newLevel
				changed = true
			}
		}
	}

	var nodes []Node
	for mod := range modules {
		group := "external"
		switch {
		case mod == "github.com/larsartmann/go-cqrs-lite/event":
			group = "core"
		case mod == "github.com/larsartmann/go-cqrs-lite/codec" ||
			mod == "github.com/larsartmann/go-cqrs-lite/otel":
			group = "infra"
			group = "test"
		case strings.HasPrefix(mod, "github.com/larsartmann/go-cqrs-lite/cmd/"):
			group = "cmd"
		case strings.HasPrefix(mod, "github.com/larsartmann/go-cqrs-lite/example/"):
			group = "example"
		case strings.HasPrefix(mod, "github.com/larsartmann/go-cqrs-lite/"):
			group = "module"
		}

		label := strings.TrimPrefix(mod, "github.com/larsartmann/go-cqrs-lite/")
		if label == "" {
			label = mod
		}

		title := fmt.Sprintf("Module: %s\\nLevel: %d", mod, levelMap[mod])

		nodes = append(nodes, Node{
			ID:    mod,
			Label: label,
			Group: group,
			Level: levelMap[mod],
			Title: title,
		})
	}

	return &GraphData{Nodes: nodes, Edges: edgeList}, nil
}

const htmlPage = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<title>go-cqrs-lite Module Graph</title>
<script src="https://unpkg.com/vis-network/standalone/umd/vis-network.min.js"></script>
<style>
*{margin:0;padding:0;box-sizing:border-box}
body{font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",Roboto,sans-serif;background:#0d1117;color:#c9d1d9}
#header{padding:16px 24px;background:#161b22;border-bottom:1px solid #30363d;display:flex;align-items:center;justify-content:space-between}
#header h1{font-size:18px;font-weight:600;color:#f0f6fc}
#header .subtitle{font-size:13px;color:#8b949e;margin-top:4px}
#graph{width:100vw;height:calc(100vh - 64px)}
#legend{position:fixed;bottom:16px;right:16px;background:#161b22;border:1px solid #30363d;border-radius:8px;padding:12px 16px;font-size:12px;box-shadow:0 4px 12px rgba(0,0,0,0.3);z-index:100}
#legend .item{display:flex;align-items:center;gap:8px;margin:4px 0}
#legend .dot{width:10px;height:10px;border-radius:50%}
#controls{position:fixed;top:72px;right:16px;display:flex;gap:8px;z-index:100}
#controls button{background:#21262d;border:1px solid #30363d;color:#c9d1d9;padding:6px 12px;border-radius:6px;cursor:pointer;font-size:12px}
#controls button:hover{background:#30363d}
#controls button.active{background:#0969da;color:#fff;border-color:#0969da}
#search{position:fixed;top:72px;left:24px;z-index:100}
#search input{background:#0d1117;border:1px solid #30363d;color:#c9d1d9;padding:6px 12px;border-radius:6px;font-size:13px;width:200px}
#search input:focus{outline:none;border-color:#58a6ff}
</style>
</head>
<body>
<div id="header">
<div>
<h1>go-cqrs-lite Module Dependency Graph</h1>
<div class="subtitle">Internal dependencies only. Arrow: depends on &rarr;</div>
</div>
</div>
<div id="search"><input type="text" id="searchInput" placeholder="Search module..." onkeyup="doSearch()"></div>
<div id="controls">
<button onclick="fit()">Fit</button>
<button id="layoutBtn" onclick="toggleLayout()">Layout: Hierarchical</button>
</div>
<div id="graph"></div>
<div id="legend">
<div class="item"><div class="dot" style="background:#58a6ff"></div> Core</div>
<div class="item"><div class="dot" style="background:#3fb950"></div> Infrastructure</div>
<div class="item"><div class="dot" style="background:#d29922"></div> Module</div>
<div class="item"><div class="dot" style="background:#a371f7"></div> Test Helpers</div>
<div class="item"><div class="dot" style="background:#f778ba"></div> Example</div>
<div class="item"><div class="dot" style="background:#8b949e"></div> Command</div>
</div>

<script>
let network, nodes, edges;
let hierarchical = true;

async function init() {
  const resp = await fetch('/api/graph');
  const data = await resp.json();

  const colors = {
    core:     { bg: '#0969da', border: '#58a6ff', fg: '#f0f6fc' },
    infra:    { bg: '#1a7f37', border: '#3fb950', fg: '#f0f6fc' },
    module:   { bg: '#7d4e00', border: '#d29922', fg: '#f0f6fc' },
    test:     { bg: '#8957e5', border: '#a371f7', fg: '#f0f6fc' },
    example:  { bg: '#9e3670', border: '#f778ba', fg: '#f0f6fc' },
    cmd:      { bg: '#484f58', border: '#8b949e', fg: '#f0f6fc' },
    external: { bg: '#21262d', border: '#484f58', fg: '#c9d1d9' },
  };

  nodes = new vis.DataSet(
    data.nodes.map(n => {
      const c = colors[n.group] || colors.external;
      return {
        id: n.id,
        label: n.label,
        group: n.group,
        level: n.level,
        title: n.title,
        color: { background: c.bg, border: c.border, highlight: { background: '#f0883e', border: '#f0883e' } },
        font: { color: c.fg, size: 13, face: 'monospace', multi: 'html' },
        shape: 'box',
        margin: 10,
        borderWidth: 2,
        shadow: { enabled: true, color: 'rgba(0,0,0,0.5)', size: 6, x: 3, y: 3 }
      };
    })
  );

  edges = new vis.DataSet(
    data.edges.map((e, i) => ({
      id: i,
      from: e.from,
      to: e.to,
      arrows: e.arrows,
      color: { color: '#484f58', highlight: '#58a6ff', hover: '#58a6ff', opacity: 0.7 },
      width: 1.5,
      smooth: { type: 'cubicBezier', forceDirection: 'horizontal', roundness: 0.4 }
    }))
  );

  const container = document.getElementById('graph');
  network = new vis.Network(container, { nodes, edges }, getOptions());

  network.on('hoverNode', function(params) {
    const nodeId = params.node;
    const connected = network.getConnectedEdges(nodeId);
    const allEdges = edges.get();
    edges.update(allEdges.map(e => {
      const isConnected = connected.includes(e.id);
      return {
        id: e.id,
        color: isConnected ? { color: '#58a6ff', opacity: 1 } : { color: '#21262d', opacity: 0.15 },
        width: isConnected ? 2.5 : 0.5
      };
    }));
  });

  network.on('blurNode', function() {
    const allEdges = edges.get();
    edges.update(allEdges.map(e => ({
      id: e.id,
      color: { color: '#484f58', opacity: 0.7 },
      width: 1.5
    })));
  });

  network.on('click', function(params) {
    if (params.nodes.length > 0) {
      const nodeId = params.nodes[0];
      console.log('Clicked:', nodeId);
    }
  });
}

function getOptions() {
  const base = {
    nodes: { font: { multi: 'html' } },
    edges: { smooth: { type: 'cubicBezier', forceDirection: 'horizontal', roundness: 0.4 } },
    interaction: { hover: true, tooltipDelay: 200, zoomView: true, dragView: true },
    physics: { enabled: !hierarchical }
  };

  if (hierarchical) {
    base.layout = {
      hierarchical: {
        enabled: true,
        direction: 'UD',
        sortMethod: 'directed',
        levelSeparation: 120,
        nodeSpacing: 180,
        treeSpacing: 200,
        blockShifting: true,
        edgeMinimization: true,
        parentCentralization: true,
        shakeTowards: 'roots'
      }
    };
    base.physics = { enabled: false };
  } else {
    base.layout = { hierarchical: { enabled: false } };
    base.physics = {
      enabled: true,
      solver: 'forceAtlas2Based',
      forceAtlas2Based: {
        gravitationalConstant: -80,
        centralGravity: 0.005,
        springLength: 200,
        springConstant: 0.18,
        damping: 0.4,
        avoidOverlap: 0.5
      },
      stabilization: { iterations: 150 }
    };
  }

  return base;
}

function fit() {
  network.fit({ animation: { duration: 500, easingFunction: 'easeInOutQuad' } });
}

function toggleLayout() {
  hierarchical = !hierarchical;
  document.getElementById('layoutBtn').textContent = hierarchical ? 'Layout: Hierarchical' : 'Layout: Physics';
  document.getElementById('layoutBtn').classList.toggle('active', hierarchical);
  network.setOptions(getOptions());
  if (hierarchical) {
    network.fit({ animation: { duration: 500 } });
  }
}

function doSearch() {
  const val = document.getElementById('searchInput').value.toLowerCase();
  if (!val) {
    network.fit();
    return;
  }
  const allNodes = nodes.get();
  const match = allNodes.find(n => n.label.toLowerCase().includes(val));
  if (match) {
    network.focus(match.id, { scale: 1.5, animation: { duration: 500 } });
    network.selectNodes([match.id]);
  }
}

init();
</script>
</body>
</html>`
