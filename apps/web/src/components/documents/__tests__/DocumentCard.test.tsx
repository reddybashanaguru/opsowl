import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { DocumentCard, Document } from "../DocumentCard";

function makeDoc(overrides: Partial<Document> = {}): Document {
  return {
    id: "doc-001",
    case_id: "case-001",
    original_filename: "evidence.pdf",
    file_type: "pdf",
    mime_type: "application/pdf",
    file_size_bytes: 204800,
    processing_status: "completed",
    page_count: 5,
    chunk_count: 12,
    error_message: null,
    uploaded_by: { id: "u1", full_name: "Alice Smith", email: "alice@test.com" },
    download_url: "http://minio/evidence.pdf",
    created_at: "2026-05-17T00:00:00Z",
    updated_at: "2026-05-17T00:00:00Z",
    ...overrides,
  };
}

describe("DocumentCard", () => {
  it("renders filename", () => {
    render(<DocumentCard document={makeDoc()} />);
    expect(screen.getByText("evidence.pdf")).toBeInTheDocument();
  });

  it("renders processing status badge", () => {
    render(<DocumentCard document={makeDoc({ processing_status: "processing" })} />);
    expect(screen.getByText("Processing")).toBeInTheDocument();
  });

  it("renders page and chunk counts", () => {
    render(<DocumentCard document={makeDoc()} />);
    expect(screen.getByText("5 pages")).toBeInTheDocument();
    expect(screen.getByText("12 chunks")).toBeInTheDocument();
  });

  it("omits page count when null", () => {
    render(<DocumentCard document={makeDoc({ page_count: null })} />);
    expect(screen.queryByText(/pages/)).not.toBeInTheDocument();
  });

  it("renders file size formatted", () => {
    render(<DocumentCard document={makeDoc({ file_size_bytes: 204800 })} />);
    expect(screen.getByText("200.0 KB")).toBeInTheDocument();
  });

  it("renders error message when present", () => {
    render(<DocumentCard document={makeDoc({ error_message: "extraction failed" })} />);
    expect(screen.getByText("extraction failed")).toBeInTheDocument();
  });

  it("does not render error message when null", () => {
    render(<DocumentCard document={makeDoc({ error_message: null })} />);
    expect(screen.queryByText("extraction failed")).not.toBeInTheDocument();
  });

  it("renders download link when url present", () => {
    render(<DocumentCard document={makeDoc()} />);
    const link = screen.getByText("Download");
    expect(link).toHaveAttribute("href", "http://minio/evidence.pdf");
  });

  it("does not render download link when url absent", () => {
    render(<DocumentCard document={makeDoc({ download_url: undefined })} />);
    expect(screen.queryByText("Download")).not.toBeInTheDocument();
  });

  it("calls onDelete with confirmation", () => {
    vi.spyOn(window, "confirm").mockReturnValue(true);
    const onDelete = vi.fn();
    render(<DocumentCard document={makeDoc()} onDelete={onDelete} />);
    fireEvent.click(screen.getByText("Delete"));
    expect(onDelete).toHaveBeenCalledWith("doc-001");
    vi.restoreAllMocks();
  });

  it("does not call onDelete when confirmation is cancelled", () => {
    vi.spyOn(window, "confirm").mockReturnValue(false);
    const onDelete = vi.fn();
    render(<DocumentCard document={makeDoc()} onDelete={onDelete} />);
    fireEvent.click(screen.getByText("Delete"));
    expect(onDelete).not.toHaveBeenCalled();
    vi.restoreAllMocks();
  });

  it("hides delete button when onDelete is not provided", () => {
    render(<DocumentCard document={makeDoc()} />);
    expect(screen.queryByText("Delete")).not.toBeInTheDocument();
  });

  it("shows pdf icon for pdf files", () => {
    const { container } = render(<DocumentCard document={makeDoc({ file_type: "pdf" })} />);
    expect(container.textContent).toContain("📄");
  });

  it("shows spreadsheet icon for xlsx files", () => {
    const { container } = render(<DocumentCard document={makeDoc({ file_type: "xlsx" })} />);
    expect(container.textContent).toContain("📊");
  });

  it("renders uploader name", () => {
    render(<DocumentCard document={makeDoc()} />);
    expect(screen.getByText(/Alice Smith/)).toBeInTheDocument();
  });

  // ── Phase 2: retry button ──────────────────────────────────────────────────

  it("shows Retry button when processing_status is failed and onRetry is provided", () => {
    const onRetry = vi.fn();
    render(
      <DocumentCard
        document={makeDoc({ processing_status: "failed" })}
        onRetry={onRetry}
      />,
    );
    expect(screen.getByText("Retry")).toBeInTheDocument();
  });

  it("does not show Retry button when processing_status is completed", () => {
    const onRetry = vi.fn();
    render(
      <DocumentCard
        document={makeDoc({ processing_status: "completed" })}
        onRetry={onRetry}
      />,
    );
    expect(screen.queryByText("Retry")).not.toBeInTheDocument();
  });

  it("does not show Retry button when onRetry is not provided even if failed", () => {
    render(<DocumentCard document={makeDoc({ processing_status: "failed" })} />);
    expect(screen.queryByText("Retry")).not.toBeInTheDocument();
  });

  it("calls onRetry with document id when Retry is clicked", () => {
    const onRetry = vi.fn();
    render(
      <DocumentCard
        document={makeDoc({ processing_status: "failed" })}
        onRetry={onRetry}
      />,
    );
    fireEvent.click(screen.getByText("Retry"));
    expect(onRetry).toHaveBeenCalledWith("doc-001");
  });

  it("shows Retrying… text and disables button when retrying is true", () => {
    const onRetry = vi.fn();
    render(
      <DocumentCard
        document={makeDoc({ processing_status: "failed" })}
        onRetry={onRetry}
        retrying
      />,
    );
    const btn = screen.getByText("Retrying…");
    expect(btn).toBeDisabled();
  });
});
