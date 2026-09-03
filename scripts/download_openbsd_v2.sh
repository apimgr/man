#!/bin/sh
# Download OpenBSD man pages using bsdtar for proper extraction

OUTPUT_DIR="${1:-/app/src/data/man}"
TEMP_DIR="/tmp/openbsd_manpages"

mkdir -p "$OUTPUT_DIR/_shared"
mkdir -p "$OUTPUT_DIR/openbsd"
mkdir -p "$TEMP_DIR"

echo "=== Downloading OpenBSD man pages ==="

# OpenBSD 7.8 man pages set
OPENBSD_URL="https://cdn.openbsd.org/pub/OpenBSD/7.8/amd64/man78.tgz"
echo "Downloading OpenBSD 7.8 man78.tgz..."
curl -L --progress-bar "$OPENBSD_URL" -o "$TEMP_DIR/man.tgz"

echo "Checking archive format..."
file "$TEMP_DIR/man.tgz"

echo "Extracting man pages..."
cd "$TEMP_DIR"
# Try multiple extraction methods
bsdtar -xf man.tgz 2>/dev/null || gzip -dc man.tgz | bsdtar -xf - 2>/dev/null || tar -xzf man.tgz 2>/dev/null || true

# Check what was extracted
echo "Looking for man pages..."
find "$TEMP_DIR" -type d -name "man*" 2>/dev/null | head -5
ls -la "$TEMP_DIR" 2>/dev/null | head -10

# Find the man directory
MANDIR=""
for dir in "$TEMP_DIR/usr/share/man" "$TEMP_DIR/./usr/share/man"; do
    if [ -d "$dir" ]; then
        MANDIR="$dir"
        break
    fi
done

if [ -z "$MANDIR" ]; then
    echo "Man directory not found, searching..."
    MANDIR=$(find "$TEMP_DIR" -type d -name "man1" 2>/dev/null | head -1 | xargs dirname 2>/dev/null)
fi

if [ -z "$MANDIR" ] || [ ! -d "$MANDIR" ]; then
    echo "ERROR: Could not find man pages directory"
    echo "Contents of temp dir:"
    find "$TEMP_DIR" -type d 2>/dev/null | head -20
    exit 1
fi

echo "Found man pages in: $MANDIR"

echo "Processing man pages..."
for section in 1 2 3 4 5 6 7 8 9; do
    mkdir -p "$OUTPUT_DIR/openbsd/$section"

    srcdir="$MANDIR/man$section"
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
