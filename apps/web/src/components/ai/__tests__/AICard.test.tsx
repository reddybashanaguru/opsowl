import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { AICard, AILog } from "../AICard";

function makeLog(overrides: Partial<AILog> = {}): AILog {
  return {
    id: "log-001",
    case_id: "case-001",
    action_type: "ask",
    response_text: "The key risk factors are: suspicious transfers and missing KYC.",
    model_used: "claude-opus-4-5",
    provider: "anthropic",
    tokens_input: 500,
    tokens_output: 150,
    latency_ms: 1200,
    confidence_score: 0.85,
    citations: [
      {
        doc_id: "doc-1",
        doc_name: "statement.pdf",
        chunk_index: 0,
        page_number: 2,
        excerpt: "Three suspicious transfers detected.",
        similarity: 0.91,
      },
    ],
    human_review_status: "pending",
    insufficient_context: false,
    triggered_by: { id: "u1", full_name: "Alice Smith", email: "alice@test.com" },
    reviewed_by: null,
    reviewed_at: null,
    human_edited_response: null,
    created_at: "2026-05-17T10:00:00Z",
    ...overrides,
  };
}

describe("AICard", () => {
  it("renders AI Generated badge", () => {
    render(<AICard log={makeLog()} />);
    expect(screen.getByText("AI Generated")).toBeInTheDocument();
  });

  it("renders action type label", () => {
    render(<AICard log={makeLog({ action_type: "summarize" })} />);
    expect(screen.getByText("Summary")).toBeInTheDocument();
  });

  it("renders response text", () => {
    render(<AICard log={makeLog()} />);
    expect(screen.getByTestId("response-text")).toHaveTextContent(
      "The key risk factors are: suspicious transfers and missing KYC."
    );
  });

  it("renders confidence bar", () => {
    render(<AICard log={makeLog({ confidence_score: 0.85 })} />);
    expect(screen.getByRole("progressbar")).toBeInTheDocument();
  });

  it("shows low confidence warning for score < 0.5", () => {
    render(<AICard log={makeLog({ confidence_score: 0.3 })} />);
    expect(screen.getByText(/Low confidence/)).toBeInTheDocument();
  });

  it("no low confidence warning for score >= 0.5", () => {
    render(<AICard log={makeLog({ confidence_score: 0.75 })} />);
    expect(screen.queryByText(/Low confidence/)).not.toBeInTheDocument();
  });

  it("renders citation count toggle", () => {
    render(<AICard log={makeLog()} />);
    expect(screen.getByText(/1 source/)).toBeInTheDocument();
  });

  it("shows citations panel on toggle click", () => {
    render(<AICard log={makeLog()} />);
    fireEvent.click(screen.getByText(/1 source/));
    expect(screen.getByTestId("citations-panel")).toBeInTheDocument();
    expect(screen.getByText("statement.pdf")).toBeInTheDocument();
  });

  it("hides citations on second toggle click", () => {
    render(<AICard log={makeLog()} />);
    fireEvent.click(screen.getByText(/1 source/));
    fireEvent.click(screen.getByText(/1 source/));
    expect(screen.queryByTestId("citations-panel")).not.toBeInTheDocument();
  });

  it("renders model and latency in metadata", () => {
    render(<AICard log={makeLog()} />);
    expect(screen.getByText("claude-opus-4-5")).toBeInTheDocument();
    expect(screen.getByText("1.2s")).toBeInTheDocument();
  });

  it("renders triggered_by name", () => {
    render(<AICard log={makeLog()} />);
    expect(screen.getByText(/Alice Smith/)).toBeInTheDocument();
  });

  it("shows review buttons when canReview and status is pending", () => {
    render(<AICard log={makeLog()} canReview={true} onApprove={vi.fn()} onReject={vi.fn()} onEdit={vi.fn()} />);
    expect(screen.getByTestId("approve-btn")).toBeInTheDocument();
    expect(screen.getByTestId("reject-btn")).toBeInTheDocument();
    expect(screen.getByTestId("edit-btn")).toBeInTheDocument();
  });

  it("hides review buttons when canReview is false", () => {
    render(<AICard log={makeLog()} canReview={false} />);
    expect(screen.queryByTestId("approve-btn")).not.toBeInTheDocument();
  });

  it("hides review buttons when status is approved", () => {
    render(<AICard log={makeLog({ human_review_status: "approved" })} canReview={true} />);
    expect(screen.queryByTestId("approve-btn")).not.toBeInTheDocument();
  });

  it("calls onApprove when approve button clicked", () => {
    const onApprove = vi.fn();
    render(<AICard log={makeLog()} canReview={true} onApprove={onApprove} onReject={vi.fn()} onEdit={vi.fn()} />);
    fireEvent.click(screen.getByTestId("approve-btn"));
    expect(onApprove).toHaveBeenCalledWith("log-001");
  });

  it("shows edit textarea on edit button click", () => {
    render(<AICard log={makeLog()} canReview={true} onApprove={vi.fn()} onReject={vi.fn()} onEdit={vi.fn()} />);
    fireEvent.click(screen.getByTestId("edit-btn"));
    expect(screen.getByTestId("edit-textarea")).toBeInTheDocument();
  });

  it("shows reject reason input on reject button click", () => {
    render(<AICard log={makeLog()} canReview={true} onApprove={vi.fn()} onReject={vi.fn()} onEdit={vi.fn()} />);
    fireEvent.click(screen.getByTestId("reject-btn"));
    expect(screen.getByTestId("reject-reason-input")).toBeInTheDocument();
  });

  it("shows edited response when human_edited_response is set", () => {
    render(<AICard log={makeLog({ human_edited_response: "Edited text by analyst." })} />);
    expect(screen.getByTestId("response-text")).toHaveTextContent("Edited text by analyst.");
  });

  it("shows reviewed_by name when present", () => {
    render(
      <AICard
        log={makeLog({
          human_review_status: "approved",
          reviewed_by: { id: "u2", full_name: "Bob Jones" },
          reviewed_at: "2026-05-17T11:00:00Z",
        })}
      />
    );
    expect(screen.getByText(/reviewed by Bob Jones/)).toBeInTheDocument();
  });

  it("renders plural sources correctly", () => {
    render(
      <AICard
        log={makeLog({
          citations: [
            { doc_id: "d1", doc_name: "a.pdf", chunk_index: 0, excerpt: "ex1", similarity: 0.8 },
            { doc_id: "d2", doc_name: "b.pdf", chunk_index: 0, excerpt: "ex2", similarity: 0.7 },
          ],
        })}
      />
    );
    expect(screen.getByText(/2 sources/)).toBeInTheDocument();
  });

  it("has data-testid ai-card", () => {
    render(<AICard log={makeLog()} />);
    expect(screen.getByTestId("ai-card")).toBeInTheDocument();
  });
});
