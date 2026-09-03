#!/bin/sh
# Download ALL man pages from official sources using package managers
# This script should be run inside appropriate containers

OUTPUT_DIR="${1:-/app/src/data/man}"
OS_NAME="${2:-linux}"

mkdir -p "$OUTPUT_DIR/_shared"
mkdir -p "$OUTPUT_DIR/$OS_NAME"

process_man_directory() {
    local mandir="$1"
    local os="$2"

    for section in 1 2 3 4 5 6 7 8 9; do
        mkdir -p "$OUTPUT_DIR/$os/$section"

        for srcdir in "$mandir/man$section" "$mandir/cat$section"; do
            [ -d "$srcdir" ] || continue

            for file in "$srcdir"/*; do
                [ -f "$file" ] || continue

                # Get base name without section suffix
                name=$(basename "$file" | sed 's/\.[0-9][a-z]*\(\.gz\)\?$//')

                # Convert to plain text
                case "$file" in
                    *.gz)
                        text=$(zcat "$file" 2>/dev/null | groff -man -Tutf8 -P-c 2>/dev/null) || text=$(zcat "$file" 2>/dev/null | col -bx 2>/dev/null) || continue
                        ;;
                    *)
                        text=$(groff -man -Tutf8 -P-c "$file" 2>/dev/null) || text=$(col -bx < "$file" 2>/dev/null) || continue
                        ;;
                esac

                [ -z "$text" ] && continue

                # Hash and deduplicate
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
    done
}

case "$OS_NAME" in
    linux)
        echo "=== Installing Linux man pages ==="
        # Update and install all man page packages
        apt-get update -qq
        apt-get install -y -qq man-db manpages manpages-dev manpages-posix manpages-posix-dev 2>/dev/null || true

        # Process system man pages
        for mandir in /usr/share/man /usr/local/share/man; do
            [ -d "$mandir" ] && process_man_directory "$mandir" "linux"
        done
        ;;

    debian)
        echo "=== Installing Debian man pages ==="
        apt-get update -qq
        apt-get install -y -qq man-db manpages manpages-dev 2>/dev/null || true
        for mandir in /usr/share/man /usr/local/share/man; do
            [ -d "$mandir" ] && process_man_directory "$mandir" "debian"
        done
        ;;

    alpine)
        echo "=== Installing Alpine man pages ==="
        apk add --no-cache mandoc man-pages man-pages-posix 2>/dev/null || true
        for mandir in /usr/share/man; do
            [ -d "$mandir" ] && process_man_directory "$mandir" "alpine"
        done
        ;;

    freebsd)
        echo "=== Processing FreeBSD man pages ==="
        for mandir in /usr/share/man /usr/local/man; do
            [ -d "$mandir" ] && process_man_directory "$mandir" "freebsd"
        done
        ;;

    *)
        echo "Unknown OS: $OS_NAME"
        exit 1
        ;;
esac

# Summary
echo ""
echo "=== Summary for $OS_NAME ==="
total=0
for section in 1 2 3 4 5 6 7 8 9; do
    count=$(ls "$OUTPUT_DIR/$OS_NAME/$section" 2>/dev/null | wc -l)
    [ "$count" -gt 0 ] && echo "Section $section: $count pages" && total=$((total + count))
done
echo "Total $OS_NAME pages: $total"
