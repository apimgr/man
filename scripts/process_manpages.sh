#!/bin/sh
# Process man pages from archive into our format

MANPAGES_DIR="${1:-/manpages}"
OUTPUT_DIR="${2:-/app/src/data/man}"

mkdir -p "$OUTPUT_DIR/_shared"
mkdir -p "$OUTPUT_DIR/linux"

for section in 1 2 3 4 5 6 7 8; do
    mkdir -p "$OUTPUT_DIR/linux/$section"

    # Process files from man/manX/ directory
    for dir in "$MANPAGES_DIR/man/man$section" "$MANPAGES_DIR/man$section"; do
        if [ -d "$dir" ]; then
            for file in "$dir"/*; do
                [ -f "$file" ] || continue

                # Check if it's a man page for this section
                case "$file" in
                    *."$section") ;;
                    *) continue ;;
                esac

                name=$(basename "$file" ."$section")

                # Convert to plain text using groff
                text=$(groff -man -Tutf8 -P-c "$file" 2>/dev/null)
                [ -z "$text" ] && continue

                # Calculate hash
                hash=$(printf '%s' "$text" | sha256sum | cut -c1-16)

                # Save to shared if new
                shared="$OUTPUT_DIR/_shared/$hash"
                if [ ! -f "$shared" ]; then
                    printf '%s\n' "$text" > "$shared"
                fi

                # Create symlink
                target="$OUTPUT_DIR/linux/$section/$name"
                rm -f "$target"
                ln -s "../../_shared/$hash" "$target"
            done
        fi
    done

    count=$(ls "$OUTPUT_DIR/linux/$section" 2>/dev/null | wc -l)
    echo "Section $section: $count pages"
done

echo "=== Summary ==="
total=$(find "$OUTPUT_DIR/linux" -type l | wc -l)
unique=$(ls "$OUTPUT_DIR/_shared" | wc -l)
echo "Total symlinks: $total"
echo "Unique files: $unique"
