"use client";

import { memo, useCallback, useEffect, useMemo, useState } from "react";
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
  AlertTriangle,
  ArrowLeftRight,
  Banknote,
  Building2,
  Download,
  Eye,
  EyeOff,
  RotateCcw,
  TrendingDown,
  TrendingUp,
  X,
} from "lucide-react";
import { formatCompact, formatCurrency } from "@/lib/utils/currency";

// ── Types ──────────────────────────────────────────────────────────────────

export interface FlowNode {
  id: string;
  label: string;
  holder_name: string;
  account_type: string;
  is_subject: boolean;
  total_in: number;
  total_out: number;
  txn_count: number;
}

export interface FlowEdge {
  id: string;
  source: string;
  target: string;
  amount: number;
  txn_count: number;
  channel: string;
  is_suspicious: boolean;
}

export interface FlowGraph {
  nodes: FlowNode[];
  edges: FlowEdge[];
  has_cycles: boolean;
  total_flow: number;
}

interface SelectedAccount extends FlowNode {
  inbound_edges: FlowEdge[];
  outbound_edges: FlowEdge[];
}

export interface FlowOfFundsGraphProps {
  graph: FlowGraph;
  currency?: string;
}

// ── Internal node data type stored in React Flow node bag ─────────────────

interface NodeData extends FlowNode {
  role: "source" | "intermediary" | "destination" | "unresolved";
  currency: string;
}

// ── Pure helpers (stable, no closures) ────────────────────────────────────

function nodeRole(n: FlowNode): NodeData["role"] {
  if (!n.label || n.label.startsWith("unresolved:")) return "unresolved";
  if (n.total_in === 0 && n.total_out > 0) return "source";
  if (n.total_out === 0 && n.total_in > 0) return "destination";
  return "intermediary";
}

const CHANNEL_LABELS: Record<string, string> = {
  neft: "NEFT", rtgs: "RTGS", imps: "IMPS", upi: "UPI",
  cash: "Cash", card: "Card", swift: "SWIFT", internal: "Internal",
};

const NODE_W = 160;
const NODE_H = 64;

function applyDagreLayout(nodes: Node[], edges: Edge[]): Node[] {
  const g = new dagre.graphlib.Graph();
  g.setGraph({ rankdir: "LR", nodesep: 50, ranksep: 100, edgesep: 25 });
  g.setDefaultEdgeLabel(() => ({}));
  nodes.forEach(n => g.setNode(n.id, { width: NODE_W, height: NODE_H }));
  edges.forEach(e => g.setEdge(e.source, e.target, { weight: 1 }));
  dagre.layout(g);
  return nodes.map(n => {
    const pos = g.node(n.id);
    return { ...n, position: { x: pos.x - NODE_W / 2, y: pos.y - NODE_H / 2 } };
  });
}

// ── Custom node — memo prevents re-renders from parent state changes ───────
// nodeTypes is defined OUTSIDE components — React Flow checks with reference
// equality; defining inside a component creates a new object each render,
// causing every node to unmount/remount on every parent state change.

const AccountNode = memo(function AccountNode({ data, selected }: NodeProps) {
  const d = (data as unknown) as NodeData;

  const borderClass =
    d.role === "source"      ? "border-green-500 border-2"        :
    d.role === "destination" ? "border-red-500 border-2"          :
    d.role === "unresolved"  ? "border-slate-600 border border-dashed" :
                               "border-amber-500/60 border";

  const bgClass =
    d.role === "source"      ? "bg-green-950/40"  :
    d.role === "destination" ? "bg-red-950/40"    :
    d.role === "unresolved"  ? "bg-slate-800/50"  :
                               "bg-amber-950/20";

  const roleColor =
    d.role === "source"      ? "text-green-400"  :
    d.role === "destination" ? "text-red-400"    :
    d.role === "unresolved"  ? "text-slate-500"  :
                               "text-amber-400";

  const roleLabel =
    d.role === "source"      ? "Source"      :
    d.role === "destination" ? "Destination" :
    d.role === "unresolved"  ? "Unresolved"  :
                               "Intermediary";

  const ringClass   = selected ? "ring-2 ring-cyan-400 ring-offset-1 ring-offset-slate-900" : "";
  const subjectRing = d.is_subject ? "ring-1 ring-cyan-500 ring-offset-1 ring-offset-slate-900" : "";

  return (
    <div
      className={[
        "flex flex-col px-3 py-2 rounded-lg w-40 shadow-lg transition-shadow hover:shadow-cyan-900/30",
        borderClass, bgClass, ringClass, subjectRing,
      ].filter(Boolean).join(" ")}
    >
      <div className="flex items-center gap-1 mb-0.5">
        {d.role === "unresolved"
          ? <Banknote size={10} className="text-slate-500 flex-shrink-0" />
          : <Building2 size={10} className={`${roleColor} flex-shrink-0`} />
        }
        <span className="text-[11px] font-semibold text-slate-200 truncate flex-1">{d.label}</span>
        {d.is_subject && (
          <span className="flex-shrink-0 text-[7px] bg-cyan-500/20 text-cyan-400 px-1 py-0.5 rounded border border-cyan-500/20 font-bold uppercase leading-none">
            Sub
          </span>
        )}
      </div>
      <span className={`text-[8px] font-medium uppercase tracking-wider mb-1 ${roleColor}`}>{roleLabel}</span>
      <div className="flex items-center justify-between text-[9px]">
        <span className="text-green-400">↑{formatCompact(d.total_in, d.currency)}</span>
        <span className="text-red-400">↓{formatCompact(d.total_out, d.currency)}</span>
      </div>
      <span className="text-[8px] text-slate-600 mt-0.5">{d.txn_count} txn{d.txn_count !== 1 ? "s" : ""}</span>
    </div>
  );
});

// Stable reference outside component scope — React Flow ref-equality check
const nodeTypes = { accountNode: AccountNode };

// ── MiniMap color — pure fn, no closures ──────────────────────────────────

function miniMapColor(n: Node): string {
  const d = (n.data as unknown) as NodeData;
  return d.role === "source"      ? "#22c55e"
       : d.role === "destination" ? "#ef4444"
       : d.role === "unresolved"  ? "#475569"
       :                            "#f59e0b";
}

// ── Account detail drawer ──────────────────────────────────────────────────

const AccountDrawer = memo(function AccountDrawer({
  account,
  currency,
  onClose,
}: {
  account: SelectedAccount;
  currency: string;
  onClose: () => void;
}) {
  const fmt = (v: number) => formatCurrency(v, currency);

  return (
    <div className="w-64 flex-shrink-0 bg-[#0f1a27] border-l border-[#1e3050] flex flex-col overflow-y-auto">
      {/* Header */}
      <div className="flex items-start justify-between px-4 pt-4 pb-3 border-b border-[#1e3050]">
        <div className="min-w-0 flex-1">
          <p className="text-sm font-semibold text-slate-200 leading-tight truncate">{account.label}</p>
          {account.holder_name && account.holder_name !== account.label && (
            <p className="text-xs text-slate-500 mt-0.5 truncate">{account.holder_name}</p>
          )}
          {account.is_subject && (
            <span className="inline-block mt-1 text-[9px] bg-cyan-500/20 text-cyan-400 px-1.5 py-0.5 rounded border border-cyan-500/20 font-bold uppercase">
              Subject Account
            </span>
          )}
        </div>
        <button
          onClick={onClose}
          aria-label="Close account drawer"
          className="ml-2 flex-shrink-0 text-slate-500 hover:text-slate-300 transition-colors p-0.5"
        >
          <X size={14} />
        </button>
      </div>

      {/* Stats grid */}
      <div className="px-4 py-3 space-y-2 text-xs border-b border-[#1e3050]">
        {[
          { label: "Account Type", value: account.account_type || "—", color: "text-slate-300" },
          { label: "Total In",     value: fmt(account.total_in),        color: "text-green-400" },
          { label: "Total Out",    value: fmt(account.total_out),        color: "text-red-400" },
          {
            label: "Net Flow",
            value: fmt(account.total_in - account.total_out),
            color: account.total_in - account.total_out >= 0 ? "text-green-400" : "text-red-400",
          },
          { label: "Transactions", value: String(account.txn_count), color: "text-slate-300" },
        ].map(({ label, value, color }) => (
          <div key={label} className="flex items-center justify-between gap-2">
            <span className="text-slate-500 flex-shrink-0">{label}</span>
            <span className={`${color} font-medium text-right truncate`}>{value}</span>
          </div>
        ))}
      </div>

      {/* Inbound flows */}
      {account.inbound_edges.length > 0 && (
        <div className="px-4 py-3 border-b border-[#1e3050]">
          <p className="flex items-center gap-1.5 text-[9px] text-slate-500 uppercase tracking-wider mb-2">
            <TrendingUp size={9} className="text-green-400" />Inbound Flows
          </p>
          <div className="space-y-1.5">
            {account.inbound_edges.map(e => (
              <div
                key={e.id}
                className={[
                  "flex items-center justify-between rounded px-2 py-1.5 text-[10px]",
                  e.is_suspicious
                    ? "bg-amber-900/25 border border-amber-500/20"
                    : "bg-slate-800/50",
                ].join(" ")}
              >
                <span className="text-slate-500 truncate flex-1 mr-2">
                  {CHANNEL_LABELS[e.channel] ?? e.channel} · {e.txn_count} txn
                </span>
                <span className={`font-semibold flex-shrink-0 ${e.is_suspicious ? "text-amber-400" : "text-green-400"}`}>
                  {fmt(e.amount)}
                </span>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Outbound flows */}
      {account.outbound_edges.length > 0 && (
        <div className="px-4 py-3">
          <p className="flex items-center gap-1.5 text-[9px] text-slate-500 uppercase tracking-wider mb-2">
            <TrendingDown size={9} className="text-red-400" />Outbound Flows
          </p>
          <div className="space-y-1.5">
            {account.outbound_edges.map(e => (
              <div
                key={e.id}
                className={[
                  "flex items-center justify-between rounded px-2 py-1.5 text-[10px]",
                  e.is_suspicious
                    ? "bg-amber-900/25 border border-amber-500/20"
                    : "bg-slate-800/50",
                ].join(" ")}
              >
                <span className="text-slate-500 truncate flex-1 mr-2">
                  {CHANNEL_LABELS[e.channel] ?? e.channel} · {e.txn_count} txn
                </span>
                <span className={`font-semibold flex-shrink-0 ${e.is_suspicious ? "text-amber-400" : "text-red-400"}`}>
                  {fmt(e.amount)}
                </span>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
});

// ── Summary strip ──────────────────────────────────────────────────────────

const SummaryStrip = memo(function SummaryStrip({
  graph,
  currency,
}: {
  graph: FlowGraph;
  currency: string;
}) {
  const flaggedCount  = useMemo(() => graph.edges.filter(e => e.is_suspicious).length, [graph.edges]);
  const totalTxns     = useMemo(() => graph.edges.reduce((s, e) => s + e.txn_count, 0), [graph.edges]);

  return (
    <div className="flex flex-wrap gap-5 px-4 py-2.5 border-b border-[#1e3050] bg-slate-900/20">
      {[
        { label: "Total Flow",   value: formatCurrency(graph.total_flow, currency), color: "text-cyan-400" },
        { label: "Accounts",     value: String(graph.nodes.length),                 color: "text-slate-300" },
        { label: "Transactions", value: String(totalTxns),                          color: "text-slate-300" },
        { label: "Suspicious",   value: String(flaggedCount),                       color: flaggedCount > 0 ? "text-amber-400" : "text-slate-600" },
        { label: "Cycles",       value: graph.has_cycles ? "Detected" : "None",    color: graph.has_cycles ? "text-red-400" : "text-slate-600" },
      ].map(({ label, value, color }) => (
        <div key={label} className="flex flex-col">
          <span className="text-[9px] text-slate-600 uppercase tracking-wider">{label}</span>
          <span className={`text-xs font-semibold ${color}`}>{value}</span>
        </div>
      ))}
    </div>
  );
});

// ── Empty state ────────────────────────────────────────────────────────────

function EmptyFlow() {
  return (
    <div className="flex flex-col items-center justify-center h-64 gap-3">
      <ArrowLeftRight size={36} className="text-slate-700" />
      <div className="text-center">
        <p className="text-sm text-slate-500">No transactions to visualize</p>
        <p className="text-xs text-slate-600 mt-1">Add or import transactions to see the flow of funds</p>
      </div>
    </div>
  );
}

// ── Inner graph — must live inside ReactFlowProvider ──────────────────────

function GraphInner({
  graph,
  currency,
  highlightSuspicious,
}: {
  graph: FlowGraph;
  currency: string;
  highlightSuspicious: boolean;
}) {
  const [nodes, setNodes, onNodesChange] = useNodesState<Node>([]);
  const [edges, setEdges, onEdgesChange] = useEdgesState<Edge>([]);
  const [selectedAccount, setSelectedAccount] = useState<SelectedAccount | null>(null);
  const { fitView } = useReactFlow();

  // Max edge amount — used for proportional stroke widths
  const maxAmt = useMemo(
    () => Math.max(...graph.edges.map(e => e.amount), 1),
    [graph.edges],
  );

  // Layout positions — only recomputed when graph TOPOLOGY changes.
  // Separating from edge styles means toggling "highlight suspicious"
  // does NOT re-run the dagre layout algorithm.
  const layoutedNodes = useMemo<Node[]>(() => {
    if (graph.nodes.length === 0) return [];
    const rfNodes: Node[] = graph.nodes.map(n => ({
      id:       n.id,
      type:     "accountNode",
      position: { x: 0, y: 0 },
      data:     { ...n, role: nodeRole(n), currency } satisfies NodeData,
    }));
    const topoEdges: Edge[] = graph.edges.map(e => ({
      id: e.id, source: e.source, target: e.target,
    }));
    return applyDagreLayout(rfNodes, topoEdges);
  }, [graph.nodes, graph.edges, currency]);

  // Edge styles — recomputed separately; does not trigger layout recalculation
  const styledEdges = useMemo<Edge[]>(() => {
    return graph.edges.map(e => {
      const isSusp  = e.is_suspicious;
      const dimmed  = highlightSuspicious && !isSusp;
      const strokeW = Math.max(1.5, Math.round((e.amount / maxAmt) * 6));
      const color   = isSusp ? "#f59e0b" : "#22d3ee";
      const dim     = "#1e3050";
      return {
        id:     e.id,
        source: e.source,
        target: e.target,
        label:  formatCompact(e.amount, currency),
        labelStyle: {
          fontSize:   9,
          fill:       dimmed ? dim : (isSusp ? "#d97706" : "#64748b"),
          fontFamily: "monospace",
        },
        labelBgStyle: { fill: "#0b1524", fillOpacity: dimmed ? 0.1 : 0.8 },
        style: {
          strokeWidth: strokeW,
          stroke:      dimmed ? dim : color,
          opacity:     dimmed ? 0.12 : 1,
        },
        markerEnd: {
          type:   MarkerType.ArrowClosed,
          width:  10,
          height: 10,
          color:  dimmed ? dim : color,
        },
        animated: isSusp && !dimmed,
      };
    });
  }, [graph.edges, highlightSuspicious, maxAmt, currency]);

  useEffect(() => { setNodes(layoutedNodes); }, [layoutedNodes, setNodes]);
  useEffect(() => { setEdges(styledEdges);   }, [styledEdges,   setEdges]);

  // fitView after layout positions settle
  useEffect(() => {
    if (layoutedNodes.length === 0) return;
    const t = setTimeout(() => fitView({ padding: 0.15, duration: 400 }), 50);
    return () => clearTimeout(t);
  }, [layoutedNodes, fitView]);

  // Node click — data extracted from React Flow node bag, no closures in node.data
  const handleNodeClick = useCallback(
    (_: React.MouseEvent, node: Node) => {
      const n = (node.data as unknown) as NodeData;
      setSelectedAccount({
        id:           n.id,
        label:        n.label,
        holder_name:  n.holder_name,
        account_type: n.account_type,
        is_subject:   n.is_subject,
        total_in:     n.total_in,
        total_out:    n.total_out,
        txn_count:    n.txn_count,
        inbound_edges:  graph.edges.filter(e => e.target === n.id),
        outbound_edges: graph.edges.filter(e => e.source === n.id),
      });
    },
    [graph.edges],
  );

  const handlePaneClick = useCallback(() => setSelectedAccount(null), []);

  if (graph.nodes.length === 0) return <EmptyFlow />;

  return (
    <div className="flex" style={{ height: 460 }}>
      <div className="flex-1 min-w-0">
        <ReactFlow
          nodes={nodes}
          edges={edges}
          onNodesChange={onNodesChange}
          onEdgesChange={onEdgesChange}
          nodeTypes={nodeTypes}
          onNodeClick={handleNodeClick}
          onPaneClick={handlePaneClick}
          fitView
          fitViewOptions={{ padding: 0.15 }}
          proOptions={{ hideAttribution: true }}
          minZoom={0.15}
          maxZoom={2.5}
          style={{ background: "#0b1524" }}
        >
          <Background color="#1e3050" gap={24} size={1} />
          <Controls
            style={{ background: "#131f30", border: "1px solid #1e3050" }}
            showInteractive={false}
          />
          <MiniMap
            style={{ background: "#0b1524", border: "1px solid #1e3050" }}
            nodeColor={miniMapColor}
            maskColor="rgba(0,0,0,0.45)"
          />
        </ReactFlow>
      </div>

      {selectedAccount && (
        <AccountDrawer
          account={selectedAccount}
          currency={currency}
          onClose={() => setSelectedAccount(null)}
        />
      )}
    </div>
  );
}

// ── Wrapper inside provider — owns export (needs DOM access) ───────────────

function GraphWithExport({
  graph,
  currency,
  highlightSuspicious,
  onToggleHighlight,
}: FlowOfFundsGraphProps & {
  highlightSuspicious: boolean;
  onToggleHighlight: () => void;
}) {
  const hasSuspicious = useMemo(() => graph.edges.some(e => e.is_suspicious), [graph.edges]);

  const handleExport = useCallback(async () => {
    const el = document.querySelector(".react-flow__renderer") as HTMLElement | null;
    if (!el) return;
    try {
      const url = await toPng(el, { backgroundColor: "#0b1524", pixelRatio: 2 });
      const a   = document.createElement("a");
      a.download = "flow-of-funds.png";
      a.href = url;
      a.click();
    } catch {
      // silent — user can screenshot if export unavailable
    }
  }, []);

  return (
    <>
      {/* Toolbar */}
      <div className="px-4 py-3 border-b border-[#1e3050] flex items-center justify-between flex-wrap gap-2">
        <div className="flex items-center gap-2.5 flex-wrap">
          <ArrowLeftRight size={14} className="text-cyan-400" />
          <span className="text-sm font-semibold text-slate-200">Flow of Funds</span>
          {graph.has_cycles && (
            <span className="flex items-center gap-1 text-[10px] px-2 py-0.5 bg-red-500/10 text-red-400 rounded-full border border-red-500/20 font-semibold">
              <RotateCcw size={9} />Cycle Detected
            </span>
          )}
        </div>
        <div className="flex items-center gap-2">
          {hasSuspicious && (
            <button
              onClick={onToggleHighlight}
              className={[
                "flex items-center gap-1.5 px-2.5 py-1.5 rounded-lg text-xs font-medium border transition-colors",
                highlightSuspicious
                  ? "bg-amber-500/15 border-amber-500/40 text-amber-400"
                  : "border-slate-600 text-slate-400 hover:border-slate-500 hover:text-slate-300",
              ].join(" ")}
            >
              {highlightSuspicious ? <EyeOff size={12} /> : <Eye size={12} />}
              {highlightSuspicious ? "Show All" : "Highlight Suspicious"}
            </button>
          )}
          <button
            onClick={handleExport}
            className="flex items-center gap-1 px-2.5 py-1.5 rounded-lg border border-slate-600 text-slate-400 hover:border-slate-500 hover:text-slate-300 text-xs font-medium transition-colors"
          >
            <Download size={12} />PNG
          </button>
        </div>
      </div>

      <SummaryStrip graph={graph} currency={currency ?? "INR"} />

      {graph.has_cycles && (
        <div className="mx-4 mt-3 flex items-center gap-2 px-3 py-2 bg-red-900/15 border border-red-500/25 rounded-lg text-red-300 text-xs">
          <AlertTriangle size={12} className="flex-shrink-0" />
          Circular flow detected — possible round-tripping or layering pattern. Suspicious paths are animated in amber.
        </div>
      )}

      <GraphInner
        graph={graph}
        currency={currency ?? "INR"}
        highlightSuspicious={highlightSuspicious}
      />

      {/* Legend */}
      {graph.nodes.length > 0 && (
        <div className="px-4 py-2.5 border-t border-[#1e3050] flex flex-wrap gap-x-5 gap-y-1">
          {[
            { cls: "bg-green-500",  label: "Source (outflow origin)" },
            { cls: "bg-amber-500",  label: "Intermediary (pass-through)" },
            { cls: "bg-red-500",    label: "Destination (inflow end)" },
            { cls: "border border-dashed border-slate-500", label: "Unresolved" },
          ].map(({ cls, label }) => (
            <div key={label} className="flex items-center gap-1.5">
              <div className={`w-2.5 h-2.5 rounded-sm flex-shrink-0 ${cls}`} />
              <span className="text-[10px] text-slate-600">{label}</span>
            </div>
          ))}
          <div className="flex items-center gap-1.5">
            <div className="w-4 h-0.5 bg-amber-400 rounded-full" />
            <span className="text-[10px] text-slate-600">Suspicious flow (animated)</span>
          </div>
          <div className="flex items-center gap-1.5">
            <div className="w-4 h-0.5 bg-cyan-400 rounded-full" />
            <span className="text-[10px] text-slate-600">Normal flow (width ∝ amount)</span>
          </div>
        </div>
      )}
    </>
  );
}

// ── Public component ───────────────────────────────────────────────────────

export function FlowOfFundsGraph({ graph: rawGraph, currency }: FlowOfFundsGraphProps) {
  const [highlightSuspicious, setHighlightSuspicious] = useState(false);

  // Guard against null slices serialized from Go nil values
  const graph: FlowGraph = {
    ...rawGraph,
    nodes: rawGraph.nodes ?? [],
    edges: rawGraph.edges ?? [],
  };

  return (
    <div className="bg-[#131f30] border border-[#1e3050] rounded-xl ring-1 ring-white/[0.03] overflow-hidden">
      <ReactFlowProvider>
        <GraphWithExport
          graph={graph}
          currency={currency}
          highlightSuspicious={highlightSuspicious}
          onToggleHighlight={() => setHighlightSuspicious(h => !h)}
        />
      </ReactFlowProvider>
    </div>
  );
}
