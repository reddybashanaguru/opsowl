import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { ExportButton } from "../ExportButton";

vi.mock("@/lib/api", () => ({
  api: {
    download: vi.fn(),
  },
}));

import { api } from "@/lib/api";
const mockDownload = api.download as ReturnType<typeof vi.fn>;

beforeEach(() => {
  mockDownload.mockReset();
});

describe("ExportButton", () => {
  it("renders with the provided label", () => {
    render(<ExportButton path="/api/v1/analytics/workload/export" filename="workload.csv" label="Export Workload" />);
    expect(screen.getByText("↓ Export Workload")).toBeInTheDocument();
  });

  it("calls api.download with the correct path and filename on click", async () => {
    mockDownload.mockResolvedValue(undefined);
    render(<ExportButton path="/api/v1/analytics/workload/export" filename="workload.csv" label="Export Workload" />);
    await userEvent.click(screen.getByRole("button"));
    expect(mockDownload).toHaveBeenCalledWith("/api/v1/analytics/workload/export", "workload.csv");
  });

  it("shows Exporting… while the download is in progress", async () => {
    let resolve!: () => void;
    mockDownload.mockReturnValue(new Promise<void>((r) => { resolve = r; }));
    render(<ExportButton path="/api/v1/analytics/risk/export" filename="risk.csv" label="Export Risk" />);
    await userEvent.click(screen.getByRole("button"));
    expect(screen.getByText("Exporting…")).toBeInTheDocument();
    resolve();
    await waitFor(() => expect(screen.getByText("↓ Export Risk")).toBeInTheDocument());
  });

  it("disables the button while exporting", async () => {
    let resolve!: () => void;
    mockDownload.mockReturnValue(new Promise<void>((r) => { resolve = r; }));
    render(<ExportButton path="/api/v1/analytics/workload/export" filename="workload.csv" label="Export Workload" />);
    const btn = screen.getByRole("button");
    await userEvent.click(btn);
    expect(btn).toBeDisabled();
    resolve();
    await waitFor(() => expect(btn).not.toBeDisabled());
  });

  it("shows error message when download fails", async () => {
    mockDownload.mockRejectedValue(new Error("network error"));
    render(<ExportButton path="/api/v1/analytics/workload/export" filename="workload.csv" label="Export Workload" />);
    await userEvent.click(screen.getByRole("button"));
    await waitFor(() => expect(screen.getByRole("alert")).toBeInTheDocument());
    expect(screen.getByText("Export failed. Please try again.")).toBeInTheDocument();
  });

  it("clears error on subsequent successful export", async () => {
    mockDownload.mockRejectedValueOnce(new Error("fail"));
    mockDownload.mockResolvedValue(undefined);
    render(<ExportButton path="/api/v1/analytics/workload/export" filename="workload.csv" label="Export Workload" />);
    await userEvent.click(screen.getByRole("button"));
    await waitFor(() => expect(screen.getByRole("alert")).toBeInTheDocument());
    await userEvent.click(screen.getByRole("button"));
    await waitFor(() => expect(screen.queryByRole("alert")).not.toBeInTheDocument());
  });
});
