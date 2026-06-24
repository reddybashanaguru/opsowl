"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import {
  ReactFlow,
  Background,
  Controls,
  MiniMap,
  useNodesState,
  useEdgesState,
  useReactFlow,
  ReactFlowProvider,
  MarkerType,
  type Node,
  type Edge,
  type NodeProps,
} from "@xyflow/react";
import "@xyflow/react/dist/style.css";
import dagre from "@dagrejs/dagre";
import { toPng } from "html-to-image";
import {
  User, Building2, AlertTriangle, Shield, Download,
  Filter, ChevronDown, X, Plus,
} from "lucide-react";

// ── Types ──────────────────────────────────────────────────────────────────

interface NetworkNode {
  id: string;
  label: string;
  entity_type: string;
  risk_level: string;
  risk_score: number;
  is_pep: boolean;
  is_sanctioned: boolean;
  is_watchlisted: boolean;
  depth: number;
  is_root: boolean;
}

interface NetworkEdge {
  id: string;
  source: string;
  target: string;
  relationship_type: string;
  strength: number;
}

interface NetworkData {
  root_id: string;
  depth: number;
  nodes: NetworkNode[];
  edges: NetworkEdge[];
  high_risk_count: number;
}

interface SelectedNode {
  id: string;
  label: string;
  entity_type: string;
  risk_level: string;
  risk_score: number;
  is_pep: boolean;
  is_sanctioned: boolean;
  is_watchlisted: boolean;
  depth: number;
  is_root: boolean;
}

interface EntityNetworkGraphProps {
  data: NetworkData;
  rootID: string;
  depth: number;
  onDepthChange: (d: number) => void;
  onAddRelationship?: () => void;
  onNavigateToSubject?: (id: string) => void;
}

// ── Constants ──────────────────────────────────────────────────────────────

const REL_LABELS: Record<string, string> = {
  OWNS_ACCOUNT:        "Owns Account",
  TRANSACTS_WITH:      "Transacts With",
  FAMILY_OF:           "Family Of",
  EMPLOYED_BY:         "Employed By",
  DIRECTOR_OF:         "Director Of",
  BENEFICIAL_OWNER_OF: "Beneficial Owner Of",
  SAME_ADDRESS:        "Same Address",
  SAME_PHONE:          "Same Phone",
  SAME_IP:             "Same IP",
  SAME_DEVICE:         "Same Device",
  GUARANTOR_OF:        "Guarantor Of",
  INTRODUCED_BY:       "Introduced By",
};

// Relationship categories for filtering
const REL_GROUPS = {
  Financial:    ["OWNS_ACCOUNT", "TRANSACTS_WITH", "BENEFICIAL_OWNER_OF", "GUARANTOR_OF"],
  Personal:     ["FAMILY_OF", "EMPLOYED_BY", "DIRECTOR_OF", "INTRODUCED_BY"],
  Inferred:     ["SAME_ADDRESS", "SAME_PHONE", "SAME_IP", "SAME_DEVICE"],
};

const EDGE_STYLE: Record<string, { stroke: string; strokeDasharray?: string; strokeWidth?: number; opacity?: number }> = {
  OWNS_ACCOUNT:        { stroke: "#60a5fa", strokeDasharray: "5,3" },
  TRANSACTS_WITH:      { stroke: "#34d399", strokeWidth: 2 },
  FAMILY_OF:           { stroke: "#a78bfa" },
  EMPLOYED_BY:         { stroke: "#94a3b8" },
  DIRECTOR_OF:         { stroke: "#94a3b8", strokeDasharray: "3,3" },
  BENEFICIAL_OWNER_OF: { stroke: "#f59e0b", strokeWidth: 2.5 },
  SAME_ADDRESS:        { stroke: "#475569", opacity: 0.5 },
  SAME_PHONE:          { stroke: "#475569", opacity: 0.5 },
  SAME_IP:             { stroke: "#475569", opacity: 0.4, strokeDasharray: "2,4" },
  SAME_DEVICE:         { stroke: "#475569", opacity: 0.4, strokeDasharray: "2,4" },
  GUARANTOR_OF:        { stroke: "#fb923c" },
  INTRODUCED_BY:       { stroke: "#e879f9", strokeDasharray: "4,2" },
};

const RISK_BORDER: Record<string, string> = {
  critical: "border-red-500",
  high:     "border-orange-400",
  medium:   "border-yellow-400",
  low:      "border-green-400",
};

const RISK_TEXT: Record<string, string> = {
  critical: "text-red-400",
  high:     "text-orange-400",
  medium:   "text-yellow-400",
  low:      "text-green-400",
};

const ALL_RELATIONSHIP_TYPES = Object.keys(REL_LABELS);

// ── Dagre Layout ───────────────────────────────────────────────────────────

const NODE_WIDTH  = 160;
const NODE_HEIGHT = 52;

function applyDagreLayout(nodes: Node[], edges: Edge[]): Node[] {
  const g = new dagre.graphlib.Graph();
  g.setGraph({ rankdir: "LR", nodesep: 55, ranksep: 90, edgesep: 20 });
  g.setDefaultEdgeLabel(() => ({}));
  nodes.forEach(n => g.setNode(n.id, { width: NODE_WIDTH, height: NODE_HEIGHT }));
  edges.forEach(e => g.setEdge(e.source, e.target));
  dagre.layout(g);
  return nodes.map(n => {
    const pos = g.node(n.id);
    return { ...n, position: { x: pos.x - NODE_WIDTH / 2, y: pos.y - NODE_HEIGHT / 2 } };
  });
}

// ── Custom Node ────────────────────────────────────────────────────────────

function EntityNode({ data, selected }: NodeProps) {
  const d = (data as unknown) as SelectedNode & { onClick: (n: SelectedNode) => void };
  const isRoot = d.is_root;
  const isSanctioned = d.is_sanctioned;
  const isPEP = d.is_pep;

  const borderClass = isRoot
    ? "border-cyan-400 border-2"
    : isSanctioned
    ? "border-red-500 border-2"
    : isPEP
    ? "border-violet-400 border-2"
    : RISK_BORDER[d.risk_level]
    ? `${RISK_BORDER[d.risk_level]} border`
    : "border-slate-600 border";

  const bgClass = isRoot
    ? "bg-cyan-900/30"
    : isSanctioned
    ? "bg-red-900/30"
    : "bg-[#131f30]";

  const ringClass = selected ? "ring-2 ring-cyan-400 ring-offset-1 ring-offset-slate-900" : "";

  return (
    <div
      onClick={() => d.onClick(d)}
      className={`relative flex items-center gap-2.5 px-3 py-2.5 rounded-lg cursor-pointer
        ${borderClass} ${bgClass} ${ringClass}
        shadow-lg hover:shadow-cyan-900/30 transition-all w-[160px]`}
    >
      {/* Entity icon */}
      <div className={`flex-shrink-0 rounded-md p-1.5 ${isRoot ? "bg-cyan-500/20" : "bg-slate-700/60"}`}>
        {d.entity_type === "company"
          ? <Building2 size={14} className={isRoot ? "text-cyan-400" : "text-slate-400"} />
          : <User size={14} className={isRoot ? "text-cyan-400" : "text-slate-400"} />
        }
      </div>

      {/* Name + badges */}
      <div className="flex-1 min-w-0">
        <div className={`text-[11px] font-semibold leading-tight truncate ${isRoot ? "text-cyan-300" : "text-slate-200"}`}>
          {d.label}
        </div>
        <div className="flex items-center gap-1 mt-0.5">
          {isRoot && (
            <span className="text-[8px] bg-cyan-500/20 text-cyan-400 px-1 py-0.5 rounded uppercase font-bold tracking-wider">Primary</span>
          )}
          {isSanctioned && (
            <span className="text-[8px] bg-red-500/20 text-red-400 px-1 py-0.5 rounded uppercase font-bold">Sanc</span>
          )}
          {isPEP && !isSanctioned && (
            <span className="text-[8px] bg-violet-500/20 text-violet-400 px-1 py-0.5 rounded uppercase font-bold">PEP</span>
          )}
          {!isRoot && !isSanctioned && !isPEP && (
            <span className={`text-[8px] ${RISK_TEXT[d.risk_level] ?? "text-slate-500"} uppercase`}>
              {d.risk_score}
            </span>
          )}
        </div>
      </div>

      {/* Sanctioned warning */}
      {isSanctioned && (
        <AlertTriangle size={12} className="flex-shrink-0 text-red-400" />
      )}
    </div>
  );
}

const nodeTypes = { entityNode: EntityNode };

// ── Inner Graph (needs ReactFlow context) ──────────────────────────────────

function GraphInner({
  data,
  rootID,
  depth,
  onDepthChange,
  onAddRelationship,
  onNavigateToSubject,
  activeRelTypes,
}: EntityNetworkGraphProps & { activeRelTypes: Set<string> }) {
  const [nodes, setNodes, onNodesChange] = useNodesState<Node>([]);
  const [edges, setEdges, onEdgesChange] = useEdgesState<Edge>([]);
  const [selectedNode, setSelectedNode] = useState<SelectedNode | null>(null);
  const { fitView } = useReactFlow();
  const flowRef = useRef<HTMLDivElement>(null);

  // Convert API data → React Flow nodes + edges, apply dagre layout
  useEffect(() => {
    if (!data || data.nodes.length === 0) {
      setNodes([]);
      setEdges([]);
      return;
    }

    const handleNodeClick = (n: SelectedNode) => setSelectedNode(n);

    const rfNodes: Node[] = data.nodes.map(n => ({
      id: n.id,
      type: "entityNode",
      position: { x: 0, y: 0 },
      data: { ...n, onClick: handleNodeClick },
      draggable: true,
    }));

    // Filter edges by active relationship types
    const filteredEdges = data.edges.filter(e => activeRelTypes.has(e.relationship_type));

    const rfEdges: Edge[] = filteredEdges.map(e => {
      const style = EDGE_STYLE[e.relationship_type] ?? { stroke: "#94a3b8" };
      return {
        id: e.id,
        source: e.source,
        target: e.target,
        label: REL_LABELS[e.relationship_type] ?? e.relationship_type,
        labelStyle: { fontSize: 9, fill: "#64748b", fontFamily: "monospace" },
        labelBgStyle: { fill: "#0f172a", fillOpacity: 0.8 },
        style: { strokeWidth: style.strokeWidth ?? 1.5, ...style },
        markerEnd: {
          type: MarkerType.ArrowClosed,
          width: 12,
          height: 12,
          color: style.stroke,
        },
        animated: e.relationship_type === "TRANSACTS_WITH",
      };
    });

    const laidOut = applyDagreLayout(rfNodes, rfEdges);
    setNodes(laidOut);
    setEdges(rfEdges);

    setTimeout(() => fitView({ padding: 0.15, duration: 400 }), 50);
  }, [data, activeRelTypes, setNodes, setEdges, fitView]);

  // PNG export
  const handleExport = useCallback(async () => {
    if (!flowRef.current) return;
    const pngUrl = await toPng(flowRef.current, {
      backgroundColor: "#0b1524",
      quality: 1,
      pixelRatio: 2,
    });
    const link = document.createElement("a");
    link.download = `entity-network-${rootID.slice(0, 8)}.png`;
    link.href = pngUrl;
    link.click();
  }, [rootID]);

  const isEmpty = data.nodes.length <= 1;

  if (isEmpty) {
    return (
      <div className="flex flex-col items-center justify-center h-64 text-slate-600">
        <Shield size={36} className="mb-3 text-slate-700" />
        <p className="text-sm text-slate-500">No entity connections recorded yet</p>
        <p className="text-xs text-slate-600 mt-1">Link related entities to build the investigation network</p>
        {onAddRelationship && (
          <button
            onClick={onAddRelationship}
            className="mt-4 flex items-center gap-1.5 px-4 py-2 bg-cyan-600 hover:bg-cyan-500 text-white rounded-lg text-xs font-medium transition-colors"
          >
            <Plus size={12} />Add First Connection
          </button>
        )}
      </div>
    );
  }

  return (
    <div className="relative flex h-[480px]">
      {/* Graph */}
      <div ref={flowRef} className="flex-1 min-w-0">
        <ReactFlow
          nodes={nodes}
          edges={edges}
          onNodesChange={onNodesChange}
          onEdgesChange={onEdgesChange}
          nodeTypes={nodeTypes}
          fitView
          fitViewOptions={{ padding: 0.15 }}
          proOptions={{ hideAttribution: true }}
          minZoom={0.2}
          maxZoom={2}
          style={{ background: "#0b1524" }}
          onPaneClick={() => setSelectedNode(null)}
        >
          <Background color="#1e3050" gap={24} size={1} />
          <Controls
            style={{ background: "#131f30", border: "1px solid #1e3050" }}
            showInteractive={false}
          />
          <MiniMap
            style={{ background: "#0b1524", border: "1px solid #1e3050" }}
            nodeColor={(n) => {
              const d = (n.data as unknown) as NetworkNode;
              if (d.is_root) return "#22d3ee";
              if (d.is_sanctioned) return "#ef4444";
              if (d.is_pep) return "#a855f7";
              return ({ critical: "#f87171", high: "#fb923c", medium: "#facc15", low: "#4ade80" })[d.risk_level] ?? "#94a3b8";
            }}
            maskColor="rgba(0,0,0,0.4)"
          />
        </ReactFlow>
      </div>

      {/* Node detail drawer */}
      {selectedNode && (
        <div className="w-64 flex-shrink-0 bg-[#131f30] border-l border-[#1e3050] p-4 overflow-y-auto">
          <div className="flex items-start justify-between mb-3">
            <h4 className="text-slate-200 text-sm font-semibold leading-tight">{selectedNode.label}</h4>
            <button onClick={() => setSelectedNode(null)} className="text-slate-500 hover:text-slate-300 ml-2 flex-shrink-0">
              <X size={14} />
            </button>
          </div>

          <div className="space-y-2 text-xs">
            <div className="flex justify-between">
              <span className="text-slate-500">Type</span>
              <span className="text-slate-300 capitalize">{selectedNode.entity_type}</span>
            </div>
            <div className="flex justify-between">
              <span className="text-slate-500">Risk Score</span>
              <span className={`font-bold ${RISK_TEXT[selectedNode.risk_level] ?? "text-slate-300"}`}>
                {selectedNode.risk_score}
              </span>
            </div>
            <div className="flex justify-between">
              <span className="text-slate-500">Risk Level</span>
              <span className={`capitalize ${RISK_TEXT[selectedNode.risk_level] ?? "text-slate-300"}`}>
                {selectedNode.risk_level}
              </span>
            </div>
            <div className="flex justify-between">
              <span className="text-slate-500">Network Hop</span>
              <span className="text-slate-300">Depth {selectedNode.depth}</span>
            </div>
          </div>

          {/* Flags */}
          <div className="flex flex-wrap gap-1.5 mt-3">
            {selectedNode.is_pep && (
              <span className="text-[9px] px-1.5 py-0.5 bg-violet-500/15 text-violet-400 rounded border border-violet-500/20 font-semibold uppercase">PEP</span>
            )}
            {selectedNode.is_sanctioned && (
              <span className="text-[9px] px-1.5 py-0.5 bg-red-500/15 text-red-400 rounded border border-red-500/20 font-semibold uppercase">Sanctioned</span>
            )}
            {selectedNode.is_watchlisted && (
              <span className="text-[9px] px-1.5 py-0.5 bg-amber-500/15 text-amber-400 rounded border border-amber-500/20 font-semibold uppercase">Watchlisted</span>
            )}
          </div>

          {/* Actions */}
          <div className="mt-4 space-y-2">
            {onNavigateToSubject && !selectedNode.is_root && (
              <button
                onClick={() => onNavigateToSubject(selectedNode.id)}
                className="w-full px-3 py-2 bg-cyan-600 hover:bg-cyan-500 text-white rounded-lg text-xs font-medium transition-colors"
              >
                Open Subject 360 →
              </button>
            )}
            {selectedNode.is_sanctioned && (
              <div className="flex items-start gap-2 p-2.5 bg-red-500/10 border border-red-500/20 rounded-lg">
                <AlertTriangle size={12} className="text-red-400 mt-0.5 flex-shrink-0" />
                <p className="text-[10px] text-red-300">This entity is on a sanctions list — escalate immediately</p>
              </div>
            )}
          </div>
        </div>
      )}
    </div>
  );
}

// ── Toolbar (outside ReactFlow context) ────────────────────────────────────

function Toolbar({
  depth,
  onDepthChange,
  highRiskCount,
  onExport,
  onAddRelationship,
  totalNodes,
  activeRelTypes,
  onRelTypeToggle,
  onResetFilters,
}: {
  depth: number;
  onDepthChange: (d: number) => void;
  highRiskCount: number;
  onExport: () => void;
  onAddRelationship?: () => void;
  totalNodes: number;
  activeRelTypes: Set<string>;
  onRelTypeToggle: (t: string) => void;
  onResetFilters: () => void;
}) {
  const [filterOpen, setFilterOpen] = useState(false);
  const filteredCount = ALL_RELATIONSHIP_TYPES.length - activeRelTypes.size;

  return (
    <div className="px-4 py-3 border-b border-[#1e3050] flex items-center justify-between flex-wrap gap-2">
      {/* Left: title + badges */}
      <div className="flex items-center gap-2 flex-wrap">
        <Shield size={14} className="text-cyan-400" />
        <span className="text-sm font-semibold text-slate-200">Entity Network</span>
        {totalNodes > 1 && (
          <span className="text-[10px] px-2 py-0.5 bg-slate-700 text-slate-400 rounded-full">
            {totalNodes} nodes
          </span>
        )}
        {highRiskCount > 0 && (
          <span className="flex items-center gap-1 text-[10px] px-2 py-0.5 bg-red-500/15 text-red-400 rounded-full border border-red-500/20 font-semibold">
            <AlertTriangle size={9} />{highRiskCount} high-risk
          </span>
        )}
        {filteredCount > 0 && (
          <span className="text-[10px] px-2 py-0.5 bg-amber-500/15 text-amber-400 rounded-full border border-amber-500/20">
            {filteredCount} type{filteredCount !== 1 ? "s" : ""} hidden
          </span>
        )}
      </div>

      {/* Right: controls */}
      <div className="flex items-center gap-2">
        {/* Depth selector */}
        <div className="flex items-center gap-1 bg-slate-800 rounded-lg p-0.5">
          {[1, 2, 3].map(d => (
            <button
              key={d}
              onClick={() => onDepthChange(d)}
              className={`px-2.5 py-1 rounded-md text-xs font-medium transition-colors ${
                depth === d
                  ? "bg-cyan-600 text-white"
                  : "text-slate-400 hover:text-slate-200"
              }`}
            >
              {d} hop{d !== 1 ? "s" : ""}
            </button>
          ))}
        </div>

        {/* Filter */}
        <div className="relative">
          <button
            onClick={() => setFilterOpen(o => !o)}
            className={`flex items-center gap-1 px-2.5 py-1.5 rounded-lg text-xs font-medium border transition-colors ${
              filteredCount > 0
                ? "border-amber-500/40 bg-amber-500/10 text-amber-400"
                : "border-slate-600 text-slate-400 hover:border-slate-500"
            }`}
          >
            <Filter size={12} />
            Filter
            <ChevronDown size={10} className={`transition-transform ${filterOpen ? "rotate-180" : ""}`} />
          </button>

          {filterOpen && (
            <div className="absolute right-0 top-full mt-1 w-64 bg-[#131f30] border border-[#1e3050] rounded-xl shadow-2xl z-50 p-3">
              <div className="flex items-center justify-between mb-2">
                <span className="text-xs text-slate-400 font-medium">Relationship Types</span>
                <button onClick={onResetFilters} className="text-[10px] text-cyan-400 hover:text-cyan-300">Reset</button>
              </div>
              {Object.entries(REL_GROUPS).map(([group, types]) => (
                <div key={group} className="mb-2">
                  <div className="text-[9px] text-slate-600 uppercase tracking-wider mb-1">{group}</div>
                  <div className="space-y-0.5">
                    {types.map(t => (
                      <label key={t} className="flex items-center gap-2 cursor-pointer group">
                        <input
                          type="checkbox"
                          checked={activeRelTypes.has(t)}
                          onChange={() => onRelTypeToggle(t)}
                          className="w-3 h-3 accent-cyan-500"
                        />
                        <span className="text-xs text-slate-400 group-hover:text-slate-200 transition-colors">
                          {REL_LABELS[t]}
                        </span>
                        <span
                          className="ml-auto w-2.5 h-1.5 rounded-sm flex-shrink-0"
                          style={{
                            backgroundColor: EDGE_STYLE[t]?.stroke ?? "#94a3b8",
                            opacity: EDGE_STYLE[t]?.opacity ?? 1,
                          }}
                        />
                      </label>
                    ))}
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>

        {/* Export */}
        <button
          onClick={onExport}
          title="Export as PNG"
          className="flex items-center gap-1 px-2.5 py-1.5 rounded-lg border border-slate-600 text-slate-400 hover:border-slate-500 hover:text-slate-200 text-xs font-medium transition-colors"
        >
          <Download size={12} />PNG
        </button>

        {/* Add relationship */}
        {onAddRelationship && (
          <button
            onClick={onAddRelationship}
            className="flex items-center gap-1 px-3 py-1.5 bg-cyan-600 hover:bg-cyan-500 text-white rounded-lg text-xs font-medium transition-colors"
          >
            <Plus size={12} />Link Entity
          </button>
        )}
      </div>
    </div>
  );
}

// ── Edge Legend ────────────────────────────────────────────────────────────

function EdgeLegend() {
  return (
    <div className="px-4 py-2 border-t border-[#1e3050] flex flex-wrap gap-x-5 gap-y-1">
      {[
        { type: "TRANSACTS_WITH", label: "Transacts With (animated)" },
        { type: "OWNS_ACCOUNT",   label: "Owns Account (dashed)" },
        { type: "BENEFICIAL_OWNER_OF", label: "Beneficial Owner" },
        { type: "SAME_ADDRESS",   label: "Inferred (low opacity)" },
      ].map(({ type, label }) => (
        <div key={type} className="flex items-center gap-1.5">
          <div
            className="w-5 h-0.5 rounded-full"
            style={{ backgroundColor: EDGE_STYLE[type]?.stroke ?? "#94a3b8" }}
          />
          <span className="text-[10px] text-slate-600">{label}</span>
        </div>
      ))}
    </div>
  );
}

// ── Root Export component ──────────────────────────────────────────────────

function GraphWithExport(props: EntityNetworkGraphProps & {
  activeRelTypes: Set<string>;
  onRelTypeToggle: (t: string) => void;
  onResetFilters: () => void;
}) {
  const { fitView } = useReactFlow();
  const flowRef = useRef<HTMLDivElement>(null);

  const handleExport = useCallback(async () => {
    const el = document.querySelector(".react-flow__renderer") as HTMLElement;
    if (!el) return;
    try {
      const pngUrl = await toPng(el, { backgroundColor: "#0b1524", pixelRatio: 2 });
      const a = document.createElement("a");
      a.download = `entity-network-${props.rootID.slice(0, 8)}.png`;
      a.href = pngUrl;
      a.click();
    } catch {
      // fallback: silent fail — user can screenshot manually
    }
  }, [props.rootID]);

  return (
    <>
      <Toolbar
        depth={props.depth}
        onDepthChange={props.onDepthChange}
        highRiskCount={props.data.high_risk_count}
        onExport={handleExport}
        onAddRelationship={props.onAddRelationship}
        totalNodes={props.data.nodes.length}
        activeRelTypes={props.activeRelTypes}
        onRelTypeToggle={props.onRelTypeToggle}
        onResetFilters={props.onResetFilters}
      />
      <GraphInner {...props} />
      {props.data.nodes.length > 1 && <EdgeLegend />}
    </>
  );
}

// ── Public component (provides ReactFlow context) ─────────────────────────

export function EntityNetworkGraph(props: EntityNetworkGraphProps) {
  const [activeRelTypes, setActiveRelTypes] = useState<Set<string>>(
    new Set(ALL_RELATIONSHIP_TYPES)
  );

  const onRelTypeToggle = useCallback((t: string) => {
    setActiveRelTypes(prev => {
      const next = new Set(prev);
      next.has(t) ? next.delete(t) : next.add(t);
      return next;
    });
  }, []);

  const onResetFilters = useCallback(() => {
    setActiveRelTypes(new Set(ALL_RELATIONSHIP_TYPES));
  }, []);

  return (
    <div className="bg-[#131f30] border border-[#1e3050] rounded-xl ring-1 ring-white/[0.04] overflow-hidden">
      <ReactFlowProvider>
        <GraphWithExport
          {...props}
          activeRelTypes={activeRelTypes}
          onRelTypeToggle={onRelTypeToggle}
          onResetFilters={onResetFilters}
        />
      </ReactFlowProvider>
    </div>
  );
}
