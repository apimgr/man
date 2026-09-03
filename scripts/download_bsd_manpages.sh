#!/bin/sh
# Download and process BSD man pages from official archives
set -e

OUTPUT_DIR="${1:-/app/src/data/man}"
TEMP_DIR="/tmp/bsd_manpages"

mkdir -p "$OUTPUT_DIR/_shared"
mkdir -p "$TEMP_DIR"

# Process man pages from a directory
process_manpages() {
    local os="$1"
    local mandir="$2"

    mkdir -p "$OUTPUT_DIR/$os"

    for section in 1 2 3 4 5 6 7 8 9; do
        mkdir -p "$OUTPUT_DIR/$os/$section"

        for dir in "$mandir/man$section" "$mandir/cat$section"; do
            [ -d "$dir" ] || continue

            for file in "$dir"/*; do
                [ -f "$file" ] || continue
                name=$(basename "$file" | sed 's/\.[0-9].*$//')

                # Skip if not a man page
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

                target="$OUTPUT_DIR/$os/$section/$name"
                rm -f "$target"
                ln -s "../../_shared/$hash" "$target"
            done
        done

        count=$(ls "$OUTPUT_DIR/$os/$section" 2>/dev/null | wc -l)
        [ "$count" -gt 0 ] && echo "[$os] Section $section: $count pages"
    done
}

# FreeBSD
echo "=== Downloading FreeBSD man pages ==="
FREEBSD_URL="https://download.freebsd.org/releases/amd64/14.2-RELEASE/base.txz"
echo "Downloading FreeBSD 14.2 base.txz..."
curl -sL "$FREEBSD_URL" -o "$TEMP_DIR/freebsd-base.txz"
echo "Extracting man pages..."
mkdir -p "$TEMP_DIR/freebsd"
tar -xf "$TEMP_DIR/freebsd-base.txz" -C "$TEMP_DIR/freebsd" --include='./usr/share/man/*' 2>/dev/null || true
process_manpages "freebsd" "$TEMP_DIR/freebsd/usr/share/man"
rm -rf "$TEMP_DIR/freebsd" "$TEMP_DIR/freebsd-base.txz"

# OpenBSD
echo ""
echo "=== Downloading OpenBSD man pages ==="
for manset in man76; do
    OPENBSD_URL="https://cdn.openbsd.org/pub/OpenBSD/7.6/amd64/${manset}.tgz"
    echo "Downloading OpenBSD 7.6 ${manset}.tgz..."
    curl -sL "$OPENBSD_URL" -o "$TEMP_DIR/openbsd-${manset}.tgz" || continue
    echo "Extracting..."
    mkdir -p "$TEMP_DIR/openbsd"
    tar -xzf "$TEMP_DIR/openbsd-${manset}.tgz" -C "$TEMP_DIR/openbsd" 2>/dev/null || true
done
process_manpages "openbsd" "$TEMP_DIR/openbsd/usr/share/man"
rm -rf "$TEMP_DIR/openbsd" "$TEMP_DIR/openbsd-"*.tgz

# NetBSD
echo ""
echo "=== Downloading NetBSD man pages ==="
NETBSD_URL="https://cdn.netbsd.org/pub/NetBSD/NetBSD-10.0/amd64/binary/sets/man.tar.xz"
echo "Downloading NetBSD 10.0 man.tar.xz..."
curl -sL "$NETBSD_URL" -o "$TEMP_DIR/netbsd-man.tar.xz"
echo "Extracting..."
mkdir -p "$TEMP_DIR/netbsd"
tar -xf "$TEMP_DIR/netbsd-man.tar.xz" -C "$TEMP_DIR/netbsd" 2>/dev/null || true
process_manpages "netbsd" "$TEMP_DIR/netbsd/usr/share/man"
rm -rf "$TEMP_DIR/netbsd" "$TEMP_DIR/netbsd-man.tar.xz"

# Summary
echo ""
echo "=== Summary ==="
for os in freebsd openbsd netbsd; do
    count=$(find "$OUTPUT_DIR/$os" -type l 2>/dev/null | wc -l)
    echo "$os: $count pages"
done
total=$(find "$OUTPUT_DIR" -type l 2>/dev/null | wc -l)
unique=$(ls "$OUTPUT_DIR/_shared" 2>/dev/null | wc -l)
echo "Total symlinks: $total"
echo "Unique files: $unique"

rm -rf "$TEMP_DIR"
