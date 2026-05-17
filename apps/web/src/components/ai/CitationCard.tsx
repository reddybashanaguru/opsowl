"use client";

export interface Citation {
  doc_id: string;
  doc_name: string;
  chunk_index: number;
  page_number?: number | null;
  excerpt: string;
  similarity: number;
}

interface CitationCardProps {
  citation: Citation;
  index: number;
}

export function CitationCard({ citation, index }: CitationCardProps) {
  const simPct = Math.round(citation.similarity * 100);

  return (
    <div className="bg-slate-800/60 border border-slate-700/50 rounded-lg p-3" data-testid="citation-card">
      <div className="flex items-center justify-between mb-1.5">
        <span className="text-xs font-medium text-violet-400">Source {index + 1}</span>
        <span className="text-xs text-slate-500">{simPct}% match</span>
      </div>
      <p className="text-xs font-medium text-slate-300 mb-1 truncate">{citation.doc_name}</p>
      {citation.page_number != null && (
        <p className="text-xs text-slate-500 mb-1">Page {citation.page_number}</p>
      )}
      <p className="text-xs text-slate-400 line-clamp-3 italic">"{citation.excerpt}"</p>
    </div>
  );
}
