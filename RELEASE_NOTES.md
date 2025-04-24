## Version 0.4

### Bug Fixes
- **Critical Fix: Review Data Processing**: Fixed a serious bug where review data was not correctly factored into spaced repetition review calculations. This improves the accuracy of card scheduling and ensures that your review history properly influences future review intervals.
- **Improved State Persistence**: Fixed issues with card state leakage, ensuring that card deletions and review updates are properly persisted between sessions.
- **Consistent FSRS State Transitions**: Updated the implementation to correctly follow the FSRS algorithm state transition behavior, ensuring reliable card scheduling.

## New Features

- **Tag Filtering for Due Cards**: Users can now filter cards by tags when retrieving due cards, allowing focused study on specific subjects
- **Available Tags Resource**: Added a new MCP resource that provides information about all available tags, including:
  - Card counts for each tag
  - Number of due cards per tag
  - Overall system statistics
  
This improves discoverability and allows users and AI assistants to see what subjects are available for study without guessing tag names.

## Binary Downloads
- flashcards: macOS/Linux build
- flashcards.exe: Windows build 