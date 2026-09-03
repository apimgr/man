#!/bin/sh
# Download OpenBSD man pages from official release archives

OUTPUT_DIR="${1:-/app/src/data/man}"
TEMP_DIR="/tmp/openbsd_manpages"

mkdir -p "$OUTPUT_DIR/_shared"
mkdir -p "$OUTPUT_DIR/openbsd"
mkdir -p "$TEMP_DIR"

echo "=== Downloading OpenBSD man pages ==="

# OpenBSD 7.6 man pages set
OPENBSD_URL="https://cdn.openbsd.org/pub/OpenBSD/7.6/amd64/man76.tgz"
echo "Downloading OpenBSD 7.6 man76.tgz..."
curl -L --progress-bar "$OPENBSD_URL" -o "$TEMP_DIR/man.tgz"

echo "Extracting man pages..."
cd "$TEMP_DIR"
# OpenBSD uses a different archive format - try multiple approaches
gzip -dc man.tgz 2>/dev/null | tar -xf - 2>/dev/null || tar -xzf man.tgz 2>/dev/null || tar -xf man.tgz 2>/dev/null || true

echo "Processing man pages..."
for section in 1 2 3 4 5 6 7 8 9; do
    mkdir -p "$OUTPUT_DIR/openbsd/$section"

    srcdir="$TEMP_DIR/usr/share/man/man$section"
    [ -d "$srcdir" ] || continue

    for file in "$srcdir"/*; do
        [ -f "$file" ] || continue

        name=$(basename "$file" | sed 's/\.[0-9][a-z]*\(\.gz\)\?$//')

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

        target="$OUTPUT_DIR/openbsd/$section/$name"
        rm -f "$target"
        ln -s "../../_shared/$hash" "$target"
    done

    count=$(ls "$OUTPUT_DIR/openbsd/$section" 2>/dev/null | wc -l)
    [ "$count" -gt 0 ] && echo "Section $section: $count pages"
done

rm -rf "$TEMP_DIR"

echo ""
echo "=== Summary ==="
total=$(find "$OUTPUT_DIR/openbsd" -type l 2>/dev/null | wc -l)
echo "Total OpenBSD pages: $total"
