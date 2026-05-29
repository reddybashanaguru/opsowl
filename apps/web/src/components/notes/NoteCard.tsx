"use client";

import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import DOMPurify from "dompurify";
import { api } from "@/lib/api";
import { useAuthStore } from "@/lib/stores/auth.store";
import { NoteEditor } from "./NoteEditor";

export interface Note {
  id: string;
  case_id: string;
  author_id: string;
  author_name: string;
  title: string;
  content: string;
  content_text: string;
  note_type: "internal" | "formal";
  is_pinned: boolean;
  ai_generated: boolean;
  ai_assisted: boolean;
  comment_count: number;
  created_at: string;
  updated_at: string;
}

interface Comment {
  id: string;
  author_id: string;
  author_name: string;
  note_id?: string;
  parent_id?: string;
  content: string;
  created_at: string;
}

interface Props {
  note: Note;
  caseId: string;
  onDeleted: () => void;
}

function CommentItem({ c }: { c: Comment }) {
  return (
    <div className="flex gap-2.5 py-2.5 border-b border-slate-700/40 last:border-0">
      <div className="w-6 h-6 rounded-full bg-slate-700 flex items-center justify-center text-[10px] font-bold text-slate-300 flex-shrink-0 mt-0.5">
        {(c.author_name ?? "?").charAt(0).toUpperCase()}
      </div>
      <div className="flex-1 min-w-0">
        <div className="flex items-center gap-2">
          <span className="text-white text-xs font-medium">{c.author_name}</span>
          <span className="text-slate-600 text-xs">
            {new Date(c.created_at).toLocaleDateString(undefined, { day: "2-digit", month: "short" })}
          </span>
        </div>
        <p className="text-slate-300 text-xs mt-0.5 leading-relaxed">{c.content}</p>
      </div>
    </div>
  );
}

export function NoteCard({ note, caseId, onDeleted }: Props) {
  const qc = useQueryClient();
  const user = useAuthStore((s) => s.user);
  const [showComments, setShowComments] = useState(false);
  const [commentText, setCommentText] = useState("");
  const [isEditing, setIsEditing] = useState(false);
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false);

  const { data: comments = [], refetch: refetchComments } = useQuery<Comment[]>({
    queryKey: ["note-comments", note.id],
    queryFn: () =>
      api.get<Comment[]>(`/api/v1/cases/${caseId}/notes/${note.id}/comments`).then(
        (r) => (r as unknown as { data: Comment[] }).data ?? r
      ),
    enabled: showComments,
  });

  const updateMutation = useMutation({
    mutationFn: (data: {
      title: string;
      content: string;
      content_text: string;
      noteType: string;
      is_pinned: boolean;
      is_human_edit: boolean;
      mentioned_user_ids: string[];
    }) =>
      api.put(`/api/v1/cases/${caseId}/notes/${note.id}`, {
        title: data.title,
        content: data.content,
        content_text: data.content_text,
        note_type: data.noteType,
        is_pinned: data.is_pinned,
        is_human_edit: data.is_human_edit,
        mentioned_user_ids: data.mentioned_user_ids,
      }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["case-notes", caseId] });
      setIsEditing(false);
    },
  });

  const pinMutation = useMutation({
    mutationFn: () =>
      api.put(`/api/v1/cases/${caseId}/notes/${note.id}`, {
        is_pinned: !note.is_pinned,
      }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["case-notes", caseId] }),
  });

  const deleteMutation = useMutation({
    mutationFn: () => api.delete(`/api/v1/cases/${caseId}/notes/${note.id}`),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["case-notes", caseId] });
      onDeleted();
    },
  });

  const commentMutation = useMutation({
    mutationFn: (content: string) =>
      api.post(`/api/v1/cases/${caseId}/comments`, {
        content,
        note_id: note.id,
      }),
    onSuccess: () => {
      setCommentText("");
      refetchComments();
      qc.invalidateQueries({ queryKey: ["case-notes", caseId] });
    },
  });

  if (isEditing) {
    return (
      <NoteEditor
        caseId={caseId}
        initialContent={note.content}
        initialTitle={note.title}
        onSave={async (data) => {
          await updateMutation.mutateAsync({
            title: data.title,
            content: data.content,
            content_text: data.contentText,
            noteType: data.noteType,
            is_pinned: note.is_pinned,
            is_human_edit: note.ai_generated,
            mentioned_user_ids: data.mentionedUserIds,
          });
        }}
        onCancel={() => setIsEditing(false)}
        isLoading={updateMutation.isPending}
        aiGenerated={note.ai_generated}
      />
    );
  }

  const isAuthor = user?.id === note.author_id;
  const canEdit = isAuthor || ["manager", "admin", "super_admin"].includes(user?.role ?? "");

  return (
    <div className={`bg-[#1e293b] border rounded-xl overflow-hidden transition-colors ${
      note.is_pinned ? "border-amber-500/40" : "border-slate-700"
    }`}>
      {/* Header */}
      <div className="px-4 pt-3.5 pb-2.5 flex items-start justify-between gap-3">
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2 flex-wrap">
            {note.is_pinned && (
              <span className="text-amber-400 text-xs font-semibold">📌 Pinned</span>
            )}
            {note.ai_generated && (
              <span className="bg-violet-900/40 text-violet-400 border border-violet-500/30 text-[10px] font-semibold px-1.5 py-0.5 rounded">
                ✨ AI Generated
              </span>
            )}
            {note.ai_assisted && !note.ai_generated && (
              <span className="bg-violet-900/20 text-violet-500 border border-violet-600/20 text-[10px] font-medium px-1.5 py-0.5 rounded">
                AI Assisted
              </span>
            )}
            {note.note_type === "formal" && (
              <span className="bg-emerald-900/30 text-emerald-400 border border-emerald-500/30 text-[10px] font-semibold px-1.5 py-0.5 rounded">
                Formal
              </span>
            )}
          </div>
          {note.title && (
            <h3 className="text-white font-semibold text-sm mt-1 leading-snug">{note.title}</h3>
          )}
          <div className="flex items-center gap-2 mt-0.5">
            <span className="text-slate-500 text-xs">{note.author_name}</span>
            <span className="text-slate-700 text-xs">·</span>
            <span className="text-slate-600 text-xs">
              {new Date(note.created_at).toLocaleDateString(undefined, {
                day: "2-digit", month: "short", year: "numeric",
              })}
            </span>
          </div>
        </div>

        {canEdit && (
          <div className="flex items-center gap-1 flex-shrink-0">
            <button
              onClick={() => pinMutation.mutate()}
              title={note.is_pinned ? "Unpin" : "Pin to top"}
              className={`text-xs px-2 py-1 rounded transition-colors ${
                note.is_pinned
                  ? "text-amber-400 hover:text-slate-400"
                  : "text-slate-600 hover:text-amber-400"
              }`}
            >
              📌
            </button>
            <button
              onClick={() => setIsEditing(true)}
              className="text-slate-600 hover:text-cyan-400 text-xs px-2 py-1 rounded transition-colors"
            >
              Edit
            </button>
            {showDeleteConfirm ? (
              <div className="flex items-center gap-1">
                <button
                  onClick={() => deleteMutation.mutate()}
                  className="text-red-400 hover:text-red-300 text-xs font-medium"
                >
                  Delete
                </button>
                <button
                  onClick={() => setShowDeleteConfirm(false)}
                  className="text-slate-500 hover:text-slate-300 text-xs"
                >
                  Cancel
                </button>
              </div>
            ) : (
              <button
                onClick={() => setShowDeleteConfirm(true)}
                className="text-slate-600 hover:text-red-400 text-xs px-2 py-1 rounded transition-colors"
              >
                Delete
              </button>
            )}
          </div>
        )}
      </div>

      {/* Content */}
      <div
        className="px-4 pb-3 text-slate-300 text-sm leading-relaxed prose prose-invert prose-sm max-w-none prose-p:my-1 prose-headings:text-slate-200 prose-code:text-cyan-300 prose-code:bg-slate-800 prose-blockquote:border-slate-600 prose-blockquote:text-slate-400"
        dangerouslySetInnerHTML={{ __html: DOMPurify.sanitize(note.content) }}
      />

      {/* Comment toggle */}
      <div className="px-4 pb-3 border-t border-slate-700/50 pt-2.5">
        <button
          onClick={() => setShowComments(!showComments)}
          className="text-xs text-slate-500 hover:text-slate-300 transition-colors flex items-center gap-1.5"
        >
          <span>💬</span>
          <span>{note.comment_count} comment{note.comment_count !== 1 ? "s" : ""}</span>
          <span className="text-slate-700">{showComments ? "▲" : "▼"}</span>
        </button>

        {showComments && (
          <div className="mt-3">
            {comments.length > 0 && (
              <div className="mb-3">
                {comments.map((c) => <CommentItem key={c.id} c={c} />)}
              </div>
            )}
            <div className="flex gap-2 items-start">
              <div className="w-6 h-6 rounded-full bg-slate-700 flex items-center justify-center text-[10px] font-bold text-slate-300 flex-shrink-0 mt-1">
                {(user?.full_name ?? "?").charAt(0).toUpperCase()}
              </div>
              <div className="flex-1 flex gap-2">
                <input
                  type="text"
                  value={commentText}
                  onChange={(e) => setCommentText(e.target.value)}
                  onKeyDown={(e) => {
                    if (e.key === "Enter" && !e.shiftKey && commentText.trim()) {
                      commentMutation.mutate(commentText.trim());
                    }
                  }}
                  placeholder="Add a comment… (Enter to send)"
                  className="flex-1 bg-slate-800 border border-slate-600 text-slate-300 text-xs rounded-lg px-3 py-1.5 focus:outline-none focus:border-cyan-500 placeholder:text-slate-600"
                />
                {commentText.trim() && (
                  <button
                    onClick={() => commentMutation.mutate(commentText.trim())}
                    disabled={commentMutation.isPending}
                    className="text-xs bg-slate-700 hover:bg-slate-600 text-slate-300 px-2.5 py-1.5 rounded-lg transition-colors disabled:opacity-50"
                  >
                    Send
                  </button>
                )}
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
