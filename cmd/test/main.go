package main

import (
	"context"
	"log"

	"github.com/k0kubun/pp/v3"
	"github.com/letieu/idea-extractor/config"
	"github.com/letieu/idea-extractor/internal/analysis"
)

func main() {
	ctx := context.Background()

	cfg, err := config.Load()
	if err != nil {
		log.Fatal("Failed to load config:", err)
	}

	anl, err := analysis.New(ctx, *cfg)
	if err != nil {
		log.Fatal(err)
	}

	res, err := anl.ExtractAnalysis(ctx, `
 I built a no BS LinkedIn - hit #1 on HackerNews

What happend
Launched on Hacker News 2 days ago and…

    🔥 450 upvotes

    💬 450 comments

    👀 17k+ visitors

    ✅ 420 signups

    📥 330 waitlist entries

All 100% bootstrapped. MVP built with React,Python MongoDB and of course Cursor ^^.

Now I’m trying to figure out:

    Do I keep it free for users and charge recruiters?

    Is this just a spike or a wedge into something much bigger?

    Should I stay bootstrapped or raise a small round to accelerate growth?

Would love to hear from other indie hackers here - what would you do?

Backgrounstory if interested:
I built Openspot out of personal frustration. I was tired of the resume black hole and the performative chaos of LinkedIn, as I wasnt able to get the internship I wanted.
That led me to building my own micro site and uploading a video resume on youtube which than got me my internship instantly...but I wondered If I can help people achieve the same much simpler.

So I build:
A public directory for people open to new opportunities.
No feed. No likes. Just clean, modern, beautiful and customizable profiles (video, audio and images optional) that help you actually stand out with unique "Behind The Profile" prompts crafted just for you.
		`)

	if err != nil {
		log.Fatal(err)
	}

	pp.Print(res)
}
