# Snapser - Bring Your Own Job Examples

Welcome to the Snapser repository for Bring Your Own Job (BYOJob) examples. A BYOJob is a small HTTP service, called a **worker**, that the [Jobs snap](https://snapser.com/docs/infra/snaps/jobs) runs across your whole user base. Snapser finds the users, sends them to your worker in batches, controls the pace, retries what fails, tracks progress, and lets you pause or resume at any time. Your worker only has to process the user IDs it is given.

Typical uses are data migrations, backfills and corrections, cleanups, and scheduled maintenance with the [Scheduler](https://snapser.com/docs/infra/snaps/scheduler) snap.

We will keep adding examples to this collection, and we welcome contributions. If you have built a useful worker, feel free to share it here!

## How it works

1. Write a worker that serves `GET /health` and `POST /batch`. See [The worker contract](https://snapser.com/docs/infra/snaps/jobs#the-worker-contract).
2. Publish it with `snapctl byojob publish --tag <tag> --path <code_root>`.
3. Create a job in the Web App and point it at your image tag.
4. Test it on a few users with `snapctl byojob job-test`, then start a full run with `snapctl byojob job-run`.

## Basic Examples
1. **[Go Example](https://github.com/snapser-community/snapser-byojobs/tree/main/basic/byojob-go)**
   Basic Go BYOJob worker. It logs and acknowledges every batch, has an optional failure mode for trying out retries and failures, and comes with a script that seeds test users in your Snapend.

## Resources
- [Jobs snap documentation](https://snapser.com/docs/infra/snaps/jobs)
- [snapctl BYO Job commands](https://snapser.com/docs/resources/cli#5-byo-job---bring-your-own-job-worker)
- [Snapctl setup tutorial](https://snapser.com/docs/guides/tutorials/setup-snapctl)
- [BYOSnap examples](https://github.com/snapser-community/snapser-byosnaps)
