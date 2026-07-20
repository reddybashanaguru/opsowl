"use client";

import { FlowOfFundsGraph, type FlowGraph } from "@/components/cases/FlowOfFundsGraph";

// ── Scenario 1: Fan-out / Smurfing (1 source → N destinations) ───────────────
// Pattern: UPI Velocity Breach — one account blasts to 14 SFB/private bank accounts
const fanOut: FlowGraph = {
  has_cycles: false,
  total_flow: 2260000,
  nodes: [
    { id: "s1", label: "AXIS0023456789", holder_name: "Rajesh Kumar Gupta", account_type: "savings", is_subject: true,  total_in: 0,       total_out: 2260000, txn_count: 14 },
    { id: "d1", label: "INDB0056789012", holder_name: "Pradeep Singh",      account_type: "savings", is_subject: false, total_in: 120000,  total_out: 0,       txn_count: 1 },
    { id: "d2", label: "KVBL0034512890", holder_name: "Anita Joshi",        account_type: "savings", is_subject: false, total_in: 120000,  total_out: 0,       txn_count: 1 },
    { id: "d3", label: "UJVN0012345678", holder_name: "Ravi Krishnan",      account_type: "savings", is_subject: false, total_in: 120000,  total_out: 0,       txn_count: 1 },
    { id: "d4", label: "AUBL0078901234", holder_name: "Deepak Mehta",       account_type: "savings", is_subject: false, total_in: 120000,  total_out: 0,       txn_count: 1 },
    { id: "d5", label: "ESAF0023456789", holder_name: "Suresh Pillai",      account_type: "savings", is_subject: false, total_in: 120000,  total_out: 0,       txn_count: 1 },
    { id: "d6", label: "HDFC0012345678", holder_name: "Arun Verma",         account_type: "savings", is_subject: false, total_in: 120000,  total_out: 0,       txn_count: 1 },
    { id: "d7", label: "ICIC0056789012", holder_name: "Priya Sharma",       account_type: "savings", is_subject: false, total_in: 120000,  total_out: 0,       txn_count: 1 },
    { id: "d8", label: "SBIN0034512890", holder_name: "Sanjay Patel",       account_type: "savings", is_subject: false, total_in: 95000,   total_out: 0,       txn_count: 1 },
    { id: "d9", label: "YESB0012345678", holder_name: "Vikas Gupta",        account_type: "savings", is_subject: false, total_in: 115000,  total_out: 0,       txn_count: 1 },
    { id: "d10",label: "KVBL0089012345", holder_name: "Neha Singh",         account_type: "savings", is_subject: false, total_in: 130000,  total_out: 0,       txn_count: 1 },
    { id: "d11",label: "INDB0067890123", holder_name: "Rohit Kumar",        account_type: "savings", is_subject: false, total_in: 155000,  total_out: 0,       txn_count: 1 },
    { id: "d12",label: "UJVN0089012345", holder_name: "Aisha Begum",        account_type: "savings", is_subject: false, total_in: 175000,  total_out: 0,       txn_count: 1 },
    { id: "d13",label: "AUBL0056789012", holder_name: "Kiran Reddy",        account_type: "savings", is_subject: false, total_in: 210000,  total_out: 0,       txn_count: 1 },
    { id: "d14",label: "HDFC0089012345", holder_name: "DCT Lending",        account_type: "current", is_subject: false, total_in: 420000,  total_out: 0,       txn_count: 1 },
  ],
  edges: [
    { id: "e1",  source: "s1", target: "d1",  amount: 120000, txn_count: 1, channel: "upi",  is_suspicious: true  },
    { id: "e2",  source: "s1", target: "d2",  amount: 120000, txn_count: 1, channel: "upi",  is_suspicious: true  },
    { id: "e3",  source: "s1", target: "d3",  amount: 120000, txn_count: 1, channel: "upi",  is_suspicious: true  },
    { id: "e4",  source: "s1", target: "d4",  amount: 120000, txn_count: 1, channel: "upi",  is_suspicious: true  },
    { id: "e5",  source: "s1", target: "d5",  amount: 120000, txn_count: 1, channel: "upi",  is_suspicious: true  },
    { id: "e6",  source: "s1", target: "d6",  amount: 120000, txn_count: 1, channel: "upi",  is_suspicious: true  },
    { id: "e7",  source: "s1", target: "d7",  amount: 120000, txn_count: 1, channel: "upi",  is_suspicious: true  },
    { id: "e8",  source: "s1", target: "d8",  amount: 95000,  txn_count: 1, channel: "upi",  is_suspicious: true  },
    { id: "e9",  source: "s1", target: "d9",  amount: 115000, txn_count: 1, channel: "upi",  is_suspicious: true  },
    { id: "e10", source: "s1", target: "d10", amount: 130000, txn_count: 1, channel: "upi",  is_suspicious: true  },
    { id: "e11", source: "s1", target: "d11", amount: 155000, txn_count: 1, channel: "upi",  is_suspicious: true  },
    { id: "e12", source: "s1", target: "d12", amount: 175000, txn_count: 1, channel: "upi",  is_suspicious: true  },
    { id: "e13", source: "s1", target: "d13", amount: 210000, txn_count: 1, channel: "upi",  is_suspicious: true  },
    { id: "e14", source: "s1", target: "d14", amount: 420000, txn_count: 1, channel: "neft", is_suspicious: true  },
  ],
};

// ── Scenario 2: Linear Chain / Layering (A → B → C → D) ─────────────────────
// Pattern: Funds moved through 3 intermediate accounts to obscure origin
const linearChain: FlowGraph = {
  has_cycles: false,
  total_flow: 5000000,
  nodes: [
    { id: "a", label: "HDFC0011111111", holder_name: "Mohammad Farouk",   account_type: "current", is_subject: true,  total_in: 0,       total_out: 5000000, txn_count: 1 },
    { id: "b", label: "ICIC0022222222", holder_name: "Sunrise Traders",   account_type: "current", is_subject: false, total_in: 5000000, total_out: 4800000, txn_count: 2 },
    { id: "c", label: "YESB0033333333", holder_name: "Global Exports Pvt",account_type: "current", is_subject: false, total_in: 4800000, total_out: 4600000, txn_count: 2 },
    { id: "d", label: "SBIN0044444444", holder_name: "Offshore Holdings", account_type: "current", is_subject: false, total_in: 4600000, total_out: 0,       txn_count: 1 },
  ],
  edges: [
    { id: "e1", source: "a", target: "b", amount: 5000000, txn_count: 1, channel: "rtgs", is_suspicious: true  },
    { id: "e2", source: "b", target: "c", amount: 4800000, txn_count: 1, channel: "rtgs", is_suspicious: true  },
    { id: "e3", source: "c", target: "d", amount: 4600000, txn_count: 1, channel: "swift",is_suspicious: true  },
  ],
};

// ── Scenario 3: Cycle / Round-Tripping (A → B → C → A) ──────────────────────
// Pattern: Funds cycled between related entities to fabricate transaction volume
const cycle: FlowGraph = {
  has_cycles: true,
  total_flow: 3600000,
  nodes: [
    { id: "a", label: "AXIS0099887766", holder_name: "Agarwal Holdings",    account_type: "current", is_subject: true,  total_in: 1200000, total_out: 1200000, txn_count: 2 },
    { id: "b", label: "KVBL0088776655", holder_name: "Mayfair Real Estate", account_type: "current", is_subject: false, total_in: 1200000, total_out: 1200000, txn_count: 2 },
    { id: "c", label: "INDB0077665544", holder_name: "Pinnacle Constructions",account_type:"current", is_subject: false, total_in: 1200000, total_out: 1200000, txn_count: 2 },
  ],
  edges: [
    { id: "e1", source: "a", target: "b", amount: 1200000, txn_count: 1, channel: "neft", is_suspicious: true },
    { id: "e2", source: "b", target: "c", amount: 1200000, txn_count: 1, channel: "neft", is_suspicious: true },
    { id: "e3", source: "c", target: "a", amount: 1200000, txn_count: 1, channel: "neft", is_suspicious: true },
  ],
};

// ── Scenario 4: Fan-in / Aggregation (N sources → 1 destination) ─────────────
// Pattern: 10 mule accounts collecting cash and funnelling into 1 master account
const fanIn: FlowGraph = {
  has_cycles: false,
  total_flow: 1800000,
  nodes: [
    { id: "m1", label: "SBIN0001112222", holder_name: "Ramesh Yadav",    account_type: "savings", is_subject: false, total_in: 0,       total_out: 180000, txn_count: 1 },
    { id: "m2", label: "SBIN0002223333", holder_name: "Suresh Yadav",    account_type: "savings", is_subject: false, total_in: 0,       total_out: 180000, txn_count: 1 },
    { id: "m3", label: "SBIN0003334444", holder_name: "Mahesh Yadav",    account_type: "savings", is_subject: false, total_in: 0,       total_out: 180000, txn_count: 1 },
    { id: "m4", label: "SBIN0004445555", holder_name: "Naresh Yadav",    account_type: "savings", is_subject: false, total_in: 0,       total_out: 180000, txn_count: 1 },
    { id: "m5", label: "SBIN0005556666", holder_name: "Dinesh Yadav",    account_type: "savings", is_subject: false, total_in: 0,       total_out: 180000, txn_count: 1 },
    { id: "m6", label: "SBIN0006667777", holder_name: "Ganesh Yadav",    account_type: "savings", is_subject: false, total_in: 0,       total_out: 180000, txn_count: 1 },
    { id: "m7", label: "SBIN0007778888", holder_name: "Kamlesh Yadav",   account_type: "savings", is_subject: false, total_in: 0,       total_out: 180000, txn_count: 1 },
    { id: "m8", label: "SBIN0008889999", holder_name: "Rajesh Yadav",    account_type: "savings", is_subject: false, total_in: 0,       total_out: 180000, txn_count: 1 },
    { id: "m9", label: "SBIN0009990000", holder_name: "Viresh Yadav",    account_type: "savings", is_subject: false, total_in: 0,       total_out: 180000, txn_count: 1 },
    { id: "m10",label: "SBIN0000001111", holder_name: "Hitesh Yadav",    account_type: "savings", is_subject: false, total_in: 0,       total_out: 180000, txn_count: 1 },
    { id: "d1", label: "HDFC0099998888", holder_name: "Cash King Finance",account_type: "current", is_subject: true,  total_in: 1800000, total_out: 0,      txn_count: 10},
  ],
  edges: [
    { id: "e1",  source: "m1",  target: "d1", amount: 180000, txn_count: 1, channel: "imps", is_suspicious: true },
    { id: "e2",  source: "m2",  target: "d1", amount: 180000, txn_count: 1, channel: "imps", is_suspicious: true },
    { id: "e3",  source: "m3",  target: "d1", amount: 180000, txn_count: 1, channel: "imps", is_suspicious: true },
    { id: "e4",  source: "m4",  target: "d1", amount: 180000, txn_count: 1, channel: "imps", is_suspicious: true },
    { id: "e5",  source: "m5",  target: "d1", amount: 180000, txn_count: 1, channel: "imps", is_suspicious: true },
    { id: "e6",  source: "m6",  target: "d1", amount: 180000, txn_count: 1, channel: "imps", is_suspicious: true },
    { id: "e7",  source: "m7",  target: "d1", amount: 180000, txn_count: 1, channel: "imps", is_suspicious: true },
    { id: "e8",  source: "m8",  target: "d1", amount: 180000, txn_count: 1, channel: "imps", is_suspicious: true },
    { id: "e9",  source: "m9",  target: "d1", amount: 180000, txn_count: 1, channel: "imps", is_suspicious: true },
    { id: "e10", source: "m10", target: "d1", amount: 180000, txn_count: 1, channel: "imps", is_suspicious: true },
  ],
};

// ── Scenario 5: Diamond / 3-Layer (1 → 2 intermediaries → 1 destination) ─────
// Pattern: Classic placement → layering → integration
// Also includes normal (non-suspicious) flows to show mixed case
const diamond: FlowGraph = {
  has_cycles: false,
  total_flow: 10000000,
  nodes: [
    { id: "src", label: "AXIS0055667788", holder_name: "Vikram Sethi",      account_type: "current", is_subject: true,  total_in: 0,        total_out: 10000000, txn_count: 2 },
    { id: "i1",  label: "INDB0011223344", holder_name: "Sethi Enterprises", account_type: "current", is_subject: false, total_in: 6000000,  total_out: 5800000,  txn_count: 4 },
    { id: "i2",  label: "YESB0055443322", holder_name: "VRS Holdings",      account_type: "current", is_subject: false, total_in: 4000000,  total_out: 3900000,  txn_count: 3 },
    { id: "legit",label:"HDFC0022334455", holder_name: "Staff Salaries A/c",account_type: "payroll", is_subject: false, total_in: 200000,   total_out: 0,        txn_count: 1 },
    { id: "dst", label: "KVBL0099887766", holder_name: "Maldives Properties",account_type:"current", is_subject: false, total_in: 9700000,  total_out: 0,        txn_count: 2 },
  ],
  edges: [
    { id: "e1", source: "src",  target: "i1",    amount: 6000000, txn_count: 1, channel: "rtgs", is_suspicious: true  },
    { id: "e2", source: "src",  target: "i2",    amount: 4000000, txn_count: 1, channel: "rtgs", is_suspicious: true  },
    { id: "e3", source: "i1",   target: "legit", amount: 200000,  txn_count: 1, channel: "neft", is_suspicious: false },
    { id: "e4", source: "i1",   target: "dst",   amount: 5800000, txn_count: 1, channel: "swift",is_suspicious: true  },
    { id: "e5", source: "i2",   target: "dst",   amount: 3900000, txn_count: 1, channel: "swift",is_suspicious: true  },
  ],
};

// ── Page ───────────────────────────────────────────────────────────────────────

const SCENARIOS = [
  {
    title: "Fan-out — Smurfing / UPI Velocity",
    subtitle: "1 source → 14 destinations",
    typology: "Structuring · Smurfing",
    description: "Subject distributes funds to many Small Finance Bank accounts in a single day to avoid detection thresholds. Classic UPI velocity breach pattern.",
    graph: fanOut,
  },
  {
    title: "Linear Chain — Layering",
    subtitle: "Source → 3 intermediaries → Sink",
    typology: "Layering · SWIFT Transfer",
    description: "Funds flow through 3 shell company accounts with each hop taking a small cut, ending in an overseas account. Each hop obscures the origin.",
    graph: linearChain,
  },
  {
    title: "Cycle — Round-Tripping",
    subtitle: "A → B → C → A",
    typology: "Round-tripping · Trade-based ML",
    description: "Three related entities transfer funds in a closed loop to fabricate transaction volume and create the appearance of legitimate business activity.",
    graph: cycle,
  },
  {
    title: "Fan-in — Cash Aggregation",
    subtitle: "10 mule accounts → 1 master",
    typology: "Mule network · Aggregation",
    description: "Ten mule accounts, each receiving sub-threshold cash deposits, funnel into a single master account controlled by the subject.",
    graph: fanIn,
  },
  {
    title: "Diamond — 3-Layer Laundering",
    subtitle: "Source → 2 intermediaries → Destination",
    typology: "Placement · Layering · Integration",
    description: "Classic three-phase laundering. Funds split across two shell companies (layering), mixed with a small legitimate payroll payment, then integrated into offshore property purchase.",
    graph: diamond,
  },
];

export default function FlowScenariosPage() {
  return (
    <div className="p-6 space-y-10 max-w-6xl mx-auto">
      <div>
        <h1 className="text-xl font-bold text-slate-100">Flow of Funds — Layout Scenarios</h1>
        <p className="text-sm text-slate-500 mt-1">
          Internal demo page · 5 AML graph topologies · Verify layout before Decentro demo
        </p>
      </div>

      {SCENARIOS.map((s, i) => (
        <div key={i} className="space-y-3">
          <div className="flex items-start justify-between">
            <div>
              <div className="flex items-center gap-3">
                <span className="text-xs font-mono text-slate-600">#{i + 1}</span>
                <h2 className="text-base font-semibold text-slate-200">{s.title}</h2>
                <span className="text-[10px] px-2 py-0.5 rounded-full bg-slate-800 text-slate-400 border border-slate-700">
                  {s.subtitle}
                </span>
              </div>
              <div className="flex items-center gap-2 mt-1 ml-6">
                <span className="text-[10px] px-2 py-0.5 rounded-full bg-amber-500/10 text-amber-400 border border-amber-500/20 font-medium">
                  {s.typology}
                </span>
                <span className="text-xs text-slate-500">{s.description}</span>
              </div>
            </div>
          </div>
          <FlowOfFundsGraph graph={s.graph} currency="INR" />
        </div>
      ))}
    </div>
  );
}
