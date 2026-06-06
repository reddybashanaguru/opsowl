#!/bin/bash
# OpsOwl AI — Demo Video Builder
# Merges the Playwright browser recording with the narration audio into a final MP4.
# Usage: bash docs/build-demo.sh

set -e

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
VIDEO_DIR="$REPO_ROOT/apps/web/test-results"
AUDIO_DIR="/tmp/opsowl_demo/audio"
OUT_DIR="$REPO_ROOT/docs"

echo "=== OpsOwl Demo Builder ==="

# Find the latest demo video (not retry)
VIDEO_FILE=$(find "$VIDEO_DIR" -name "video.webm" -path "*Narrated*" | grep -v retry | sort -t/ -k1 | tail -1)
if [ -z "$VIDEO_FILE" ]; then
  VIDEO_FILE=$(find "$VIDEO_DIR" -name "video.webm" | grep -v retry | tail -1)
fi

if [ -z "$VIDEO_FILE" ]; then
  echo "ERROR: No video.webm found. Run the Playwright demo recording first."
  exit 1
fi

echo "Video:  $VIDEO_FILE"
VIDEO_DUR=$(ffprobe -v quiet -show_entries format=duration -of csv=p=0 "$VIDEO_FILE" 2>/dev/null)
echo "Duration: ${VIDEO_DUR}s"

# Concatenate all narration audio segments in order
CONCAT_LIST="/tmp/opsowl_demo/concat.txt"
> "$CONCAT_LIST"
for scene in s01_login s02_dashboard s03_cases s04_case_detail s05_tasks \
             s06_documents s07_notes s08_timeline s09_copilot s10_ai_logs \
             s11_sla s12_audit s13_analyst_divya s14_manager_ravi s15_users \
             s16_search s17_close; do
  AIFF="$AUDIO_DIR/${scene}.aiff"
  if [ -f "$AIFF" ]; then
    echo "file '$AIFF'" >> "$CONCAT_LIST"
  else
    echo "WARNING: Missing $AIFF"
  fi
done

echo "Concatenating narration audio..."
CONCAT_AUDIO="/tmp/opsowl_demo/full_narration.aac"
ffmpeg -y -f concat -safe 0 -i "$CONCAT_LIST" \
  -c:a aac -b:a 192k "$CONCAT_AUDIO" 2>/dev/null
AUDIO_DUR=$(ffprobe -v quiet -show_entries format=duration -of csv=p=0 "$CONCAT_AUDIO" 2>/dev/null)
echo "Narration: ${AUDIO_DUR}s"

# Output path
TIMESTAMP=$(date +%Y%m%d_%H%M)
OUTPUT="$OUT_DIR/OpsOwl_Demo_${TIMESTAMP}.mp4"

echo "Merging video + narration → $OUTPUT"

# Merge strategy:
# - Take video track from Playwright recording (scaled to 1440x900)
# - Use concatenated narration as audio track
# - Trim both to the shorter of the two durations (should be ~equal)
ffmpeg -y \
  -i "$VIDEO_FILE" \
  -i "$CONCAT_AUDIO" \
  -vf "scale=1440:900:force_original_aspect_ratio=decrease,pad=1440:900:(ow-iw)/2:(oh-ih)/2" \
  -c:v libx264 -preset medium -crf 20 \
  -c:a aac -b:a 192k \
  -shortest \
  -movflags +faststart \
  "$OUTPUT" 2>/dev/null

FINAL_DUR=$(ffprobe -v quiet -show_entries format=duration -of csv=p=0 "$OUTPUT" 2>/dev/null)
SIZE=$(du -h "$OUTPUT" | cut -f1)

echo ""
echo "=== BUILD COMPLETE ==="
echo "File:     $OUTPUT"
echo "Duration: ${FINAL_DUR}s ($(echo "$FINAL_DUR / 60" | bc)m)"
echo "Size:     $SIZE"
echo ""
echo "Open with:"
echo "  open \"$OUTPUT\""
