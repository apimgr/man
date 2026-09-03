#!/bin/sh
# Download NetBSD man pages from official release archives

OUTPUT_DIR="${1:-/app/src/data/man}"
TEMP_DIR="/tmp/netbsd_manpages"

mkdir -p "$OUTPUT_DIR/_shared"
mkdir -p "$OUTPUT_DIR/netbsd"
mkdir -p "$TEMP_DIR"

echo "=== Downloading NetBSD man pages ==="

# NetBSD 10.0 man pages set
NETBSD_URL="https://cdn.netbsd.org/pub/NetBSD/NetBSD-10.0/amd64/binary/sets/man.tar.xz"
echo "Downloading NetBSD 10.0 man.tar.xz..."
curl -L --progress-bar "$NETBSD_URL" -o "$TEMP_DIR/man.tar.xz"

echo "Extracting man pages..."
cd "$TEMP_DIR"
tar -xJf man.tar.xz

echo "Processing man pages..."
for section in 1 2 3 4 5 6 7 8 9; do
    mkdir -p "$OUTPUT_DIR/netbsd/$section"

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

        target="$OUTPUT_DIR/netbsd/$section/$name"
        rm -f "$target"
        ln -s "../../_shared/$hash" "$target"
    done

    count=$(ls "$OUTPUT_DIR/netbsd/$section" 2>/dev/null | wc -l)
    [ "$count" -gt 0 ] && echo "Section $section: $count pages"
done

rm -rf "$TEMP_DIR"

echo ""
echo "=== Summary ==="
total=$(find "$OUTPUT_DIR/netbsd" -type l 2>/dev/null | wc -l)
echo "Total NetBSD pages: $total"
