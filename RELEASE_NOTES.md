## Bug Fixes (0.3.1)

- **Prevent Repetition of Recently Reviewed Cards**: Fixed an issue where the same card would show up immediately after submission even when scored highly. The system now ensures a minimum interval between reviews of the same card.

## New Features (0.3.0)

- **Tag Filtering for Due Cards**: Users can now filter cards by tags when retrieving due cards, allowing focused study on specific subjects
- **Available Tags Resource**: Added a new MCP resource that provides information about all available tags, including:
  - Card counts for each tag
  - Number of due cards per tag
  - Overall system statistics
  
This improves discoverability and allows users and AI assistants to see what subjects are available for study without guessing tag names.

## Binary Downloads
- flashcards: macOS/Linux build
- flashcards.exe: Windows build 