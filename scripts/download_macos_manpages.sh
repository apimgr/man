#!/bin/sh
# Download and process macOS/Darwin man pages from Apple Open Source
set -e

OUTPUT_DIR="${1:-/app/src/data/man}"
TEMP_DIR="/tmp/macos_manpages"

mkdir -p "$OUTPUT_DIR/_shared"
mkdir -p "$OUTPUT_DIR/macos"
mkdir -p "$TEMP_DIR"

# Process man pages from a directory
process_manpages() {
    local mandir="$1"

    for section in 1 2 3 4 5 6 7 8 9; do
        mkdir -p "$OUTPUT_DIR/macos/$section"

        for dir in "$mandir/man$section" "$mandir/cat$section"; do
            [ -d "$dir" ] || continue

            for file in "$dir"/*; do
                [ -f "$file" ] || continue
                name=$(basename "$file" | sed 's/\.[0-9].*$//')

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

                target="$OUTPUT_DIR/macos/$section/$name"
                rm -f "$target"
                ln -s "../../_shared/$hash" "$target"
            done
        done

        count=$(ls "$OUTPUT_DIR/macos/$section" 2>/dev/null | wc -l)
        [ "$count" -gt 0 ] && echo "[macos] Section $section: $count pages"
    done
}

echo "=== Downloading macOS/Darwin man pages ==="

# Download from Apple Open Source - multiple packages contain man pages
PACKAGES="
shell_cmds
file_cmds
text_cmds
system_cmds
basic_cmds
developer_cmds
network_cmds
adv_cmds
"

for pkg in $PACKAGES; do
    echo "Fetching $pkg..."
    # Try to get the latest version from GitHub mirror
    url="https://github.com/apple-oss-distributions/${pkg}/archive/refs/heads/main.tar.gz"
    if curl -sL --fail -o "$TEMP_DIR/${pkg}.tar.gz" "$url" 2>/dev/null; then
        mkdir -p "$TEMP_DIR/${pkg}"
        tar -xzf "$TEMP_DIR/${pkg}.tar.gz" -C "$TEMP_DIR/${pkg}" --strip-components=1 2>/dev/null || true

        # Find and process man pages in this package
        for mandir in $(find "$TEMP_DIR/${pkg}" -type d -name "man*" 2>/dev/null); do
            case "$mandir" in
                */man[1-9])
                    section=$(basename "$mandir" | sed 's/man//')
                    mkdir -p "$OUTPUT_DIR/macos/$section"
                    for file in "$mandir"/*; do
                        [ -f "$file" ] || continue
                        name=$(basename "$file" | sed 's/\.[0-9].*$//')

                        text=$(groff -man -Tutf8 -P-c "$file" 2>/dev/null) || continue
                        [ -z "$text" ] && continue

                        hash=$(printf '%s' "$text" | sha256sum | cut -c1-16)
                        shared="$OUTPUT_DIR/_shared/$hash"

                        if [ ! -f "$shared" ]; then
                            printf '%s\n' "$text" > "$shared"
                        fi

                        target="$OUTPUT_DIR/macos/$section/$name"
                        rm -f "$target"
                        ln -s "../../_shared/$hash" "$target"
                    done
                    ;;
            esac
        done

        rm -rf "$TEMP_DIR/${pkg}" "$TEMP_DIR/${pkg}.tar.gz"
    else
        echo "  Failed to download $pkg"
    fi
done

# Also try to get xnu (kernel) man pages
echo "Fetching xnu (kernel)..."
if curl -sL --fail -o "$TEMP_DIR/xnu.tar.gz" "https://github.com/apple-oss-distributions/xnu/archive/refs/heads/main.tar.gz" 2>/dev/null; then
    mkdir -p "$TEMP_DIR/xnu"
    tar -xzf "$TEMP_DIR/xnu.tar.gz" -C "$TEMP_DIR/xnu" --strip-components=1 2>/dev/null || true

    for mandir in $(find "$TEMP_DIR/xnu" -type d -name "man*" 2>/dev/null); do
        case "$mandir" in
            */man[1-9])
                section=$(basename "$mandir" | sed 's/man//')
                mkdir -p "$OUTPUT_DIR/macos/$section"
                for file in "$mandir"/*; do
                    [ -f "$file" ] || continue
                    name=$(basename "$file" | sed 's/\.[0-9].*$//')

                    text=$(groff -man -Tutf8 -P-c "$file" 2>/dev/null) || continue
                    [ -z "$text" ] && continue

                    hash=$(printf '%s' "$text" | sha256sum | cut -c1-16)
                    shared="$OUTPUT_DIR/_shared/$hash"

                    if [ ! -f "$shared" ]; then
                        printf '%s\n' "$text" > "$shared"
                    fi

                    target="$OUTPUT_DIR/macos/$section/$name"
                    rm -f "$target"
                    ln -s "../../_shared/$hash" "$target"
                done
                ;;
        esac
    done
    rm -rf "$TEMP_DIR/xnu" "$TEMP_DIR/xnu.tar.gz"
fi

# Summary
echo ""
echo "=== Summary ==="
for section in 1 2 3 4 5 6 7 8 9; do
    count=$(ls "$OUTPUT_DIR/macos/$section" 2>/dev/null | wc -l)
    [ "$count" -gt 0 ] && echo "Section $section: $count pages"
done
total=$(find "$OUTPUT_DIR/macos" -type l 2>/dev/null | wc -l)
echo "Total macOS pages: $total"

rm -rf "$TEMP_DIR"
