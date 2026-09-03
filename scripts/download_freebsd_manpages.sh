#!/bin/sh
# Download FreeBSD man pages from official release archives

OUTPUT_DIR="${1:-/app/src/data/man}"
TEMP_DIR="/tmp/freebsd_manpages"

mkdir -p "$OUTPUT_DIR/_shared"
mkdir -p "$OUTPUT_DIR/freebsd"
mkdir -p "$TEMP_DIR"

echo "=== Downloading FreeBSD man pages ==="

# FreeBSD 14.2-RELEASE base.txz contains /usr/share/man (XZ compressed)
FREEBSD_URL="https://download.freebsd.org/releases/amd64/14.2-RELEASE/base.txz"
echo "Downloading FreeBSD 14.2 base.txz..."
curl -L --progress-bar "$FREEBSD_URL" -o "$TEMP_DIR/base.txz"

echo "Extracting man pages from archive..."
cd "$TEMP_DIR"
# Extract only usr/share/man directory (XZ format requires -J flag)
xz -d -k base.txz 2>/dev/null || true
tar -xf base.tar ./usr/share/man 2>/dev/null || tar -xJf base.txz ./usr/share/man 2>/dev/null || true
# If extraction failed, try without path filter
if [ ! -d "$TEMP_DIR/usr/share/man" ]; then
    echo "Trying full extraction..."
    tar -xJf base.txz 2>/dev/null || xz -dc base.txz | tar -xf - 2>/dev/null || true
fi

echo "Processing man pages..."
for section in 1 2 3 4 5 6 7 8 9; do
    mkdir -p "$OUTPUT_DIR/freebsd/$section"

    srcdir="$TEMP_DIR/usr/share/man/man$section"
    [ -d "$srcdir" ] || continue

    for file in "$srcdir"/*; do
        [ -f "$file" ] || continue

        name=$(basename "$file" | sed 's/\.[0-9][a-z]*\(\.gz\)\?$//')

        # Convert to plain text
        case "$file" in
            *.gz)
                text=$(zcat "$file" 2>/dev/null | groff -man -Tutf8 -P-c 2>/dev/null) || continue
                ;;
            *)
                text=$(groff -man -Tutf8 -P-c "$file" 2>/dev/null) || continue
                ;;
        esac

        [ -z "$text" ] && continue

        hash=$(printf '%s' "$text" | sha256sum | cut -c1-16)
        shared="$OUTPUT_DIR/_shared/$hash"

        if [ ! -f "$shared" ]; then
            printf '%s\n' "$text" > "$shared"
        fi

        target="$OUTPUT_DIR/freebsd/$section/$name"
        rm -f "$target"
        ln -s "../../_shared/$hash" "$target"
    done

    count=$(ls "$OUTPUT_DIR/freebsd/$section" 2>/dev/null | wc -l)
    [ "$count" -gt 0 ] && echo "Section $section: $count pages"
done

# Cleanup
rm -rf "$TEMP_DIR"

# Summary
echo ""
echo "=== Summary ==="
total=$(find "$OUTPUT_DIR/freebsd" -type l 2>/dev/null | wc -l)
echo "Total FreeBSD pages: $total"
