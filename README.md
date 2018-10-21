# StandardNoteCli
A command line interface for [StandardNote](https://standardnotes.org/)

(Work in progress. Use at your own risk)

Can be used to:
1. Sync your notes with text files in a local directory
2. Keep a bolt database synced with a standard file server for backup purposes. With a tool like Rustic you can revert back to your lates snapshot if something goes wrong.

### Usage
The command bellow will create a Notes directory and create text files for all your notes. If you change one of these while the program is running it will be synced back to the StandardNotes server.

```
./StandardNote -email *Your Email* -password *Your password*
```
