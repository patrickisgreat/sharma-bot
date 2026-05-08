# Limited Supply: MMM with Mike True

Welcome back to another episode of Limited Supply. Today we're discussing MMM, which stands for Media Mixed Modeling. We've previously covered MTA and incrementality, and today we're exploring the third pillar of modern attribution measurement.

Mike True is the creator of Prescient, an MMM tool used by many of the largest direct-to-consumer and consumer brands. He's an incredibly talented operator with deep expertise in this space, and we've broken down complex concepts into easily understandable explanations.

If you run media on view-through channels like TV, TikTok, AppLovin, or YouTube, or if you're selling in places like retail and Amazon beyond your own website, this episode is essential listening. Understanding MMM can transform how you allocate marketing budgets and measure the true impact of your media spend.

## What is MMM?

MMM stands for Marketing Mixed Model, and the concept has been around since the 1960s. It was the primary form of measurement for traditional media like billboards, catalogs, TV, and newspapers, when companies wanted to understand how their spend was driving retail sales.

The easiest way to think about MMM today is that it helps you measure your top-of-funnel spend. It takes credit from your last-click channels and redistributes it back up to campaigns where you have the highest level of confidence that the credit truly belongs. It also provides budget optimization and recommendation models for how you should shift your spend going forward.

## Mike's Background and the Origin of Prescient

Mike spent much of his career working on large-scale software transactions in the analytics space. He was on the IBM Watson team that famously beat Jeopardy in 2014, working primarily in healthcare and financial services.

However, his journey to Prescient began in an unexpected place. At Byron Bay's Blues Festival in Australia, Mike decided he wanted to build a platform to help record labels predict where artists should tour and recommend local venues. But when COVID hit and touring stopped, he had to pivot his thinking.

One of his advisors pointed out that with everyone stuck inside, record labels would shift their strategies toward awareness and earned media to drive streaming. The problem was there's no Google Analytics behind Spotify and Apple Music, so there's no last-click attribution data.

Music turned out to be the most difficult industry to measure. Think about all the variables: an artist appearing on a late-night show, Twitter live streams, viral dance trends on TikTok, plus spending across seven or eight digital advertising channels simultaneously. There's an enormous halo effect that's nearly impossible to quantify with traditional attribution.

This challenge led Mike and his team to develop statistical models that could handle this complexity. They started with Cardi B's "Up" song in February 2021. There was no tech platform available at the time—they would simply collect data, run it through their models, and provide reports to record label executives. When they recommended shifting spend from Spotify to YouTube and Facebook to Snapchat for the song's promotion, they predicted 96.3 percent accuracy in resulting streams. That was the aha moment that convinced them to build Prescient as a software platform.

## The Transition to E-commerce

When Mike shifted from the music industry to consumer brands, the core principles remained the same, but the application evolved. Your spend doesn't drive 100 percent of your revenue. There's seasonality, promotional periods, word-of-mouth, and organic demand. MMMs use historical data to quantify how much of your revenue comes from seasonality versus organic demand versus paid spend.

If your brand generated $10 million last year, an MMM will analyze your historical data and determine that perhaps 65 percent came from your paid spend, with the rest coming from other factors. That paid spend then fights for attribution credit across your various channels.

To build an effective MMM, you need several data inputs: spend by day across all channels, impression data and GA session-level conversions from both paid and organic sources, revenue data from your Shopify store, Amazon, and retail channels, and all the different ad channels you use—from connected TV to podcasts to Reddit and Google.

Things happen that you can't predict, so models will sometimes be wrong. You can't predict COVID or tariffs in advance. But what you can do is tell the model when these events occur, and it can then quantify their impact alongside your media spend impact.

## Understanding Attribution Options

When explaining attribution to those unfamiliar with the space, it helps to understand the differences between MTA, incrementality testing, MMM, and in-platform data.

MTA uses deterministic tracking based on IP addresses and clicks. It says definitively: this person took these actions across these campaigns. This approach works well for bottom-funnel, click-based channels but falls short for view-based channels like linear TV, connected TV, podcasts, YouTube, and TikTok.

Incrementality testing involves shutting off spend on a channel and measuring the impact. It tells you whether a channel is currently incremental, but it doesn't tell you what to do next. You have to run the test again, recalibrate, and repeat.

MMM uses statistical models analyzing historical data to find the optimal spending sweet spot and forecast predicted impact if you make changes. Historically, MMM models ran once or twice a year. Modern implementations can run monthly, weekly, or even daily.

All three approaches work together in a unified measurement strategy. The key insight is that there's no single source of truth—there are multiple forms of measurement designed to tell you different things for different marketing mixes. A brand spending primarily on Shopify, Meta, and Google probably doesn't need MMM yet. But once you add upper-funnel channels like YouTube, Amazon, or retail, MMM becomes valuable.

## Integrating Multiple Data Sources

Modern MMMs can incorporate incrementality data and holdout test results from platforms like House, which runs holdout tests to measure cost per incremental acquisition. If a brand is confident in their holdout results, they can input these as "priors"—prior beliefs that inform the model.

Some brands also input post-purchase survey results. Others use results from other MMMs like Google's Meridian or Robin, both open-source tools. This flexibility means marketers can triangulate between different measurement sources rather than being locked into a single approach.

The philosophy should be one of transparency and flexibility. If you have three different measurements showing 2.8, 3.2, and 3.4, and you're unsure which to trust, upcoming features in tools like Prescient can help recommend which measurement is best suited for your specific model. The power of decision-making stays with the marketer.

## Real-World Use Cases

One major use case is quantifying halo effects. When a brand diversifies its revenue mix by entering marketplaces like Amazon, last-click attribution inflates Amazon's credit because it doesn't see what's driving people to the marketplace. MMM can show how a YouTube campaign impacts Amazon sales, providing validation that the halo effect is real.

The same applies to retail. When brands sell through Costco, Whole Foods, Sephora, Target, and Walmart alongside their DTC channel, they finally get clarity on how their media spend is driving retail sales. Prescient is currently designing partnerships with Shopify POS to quantify these halo effects for retailers.

Another practical use case is scenario planning. When tariff uncertainty hits or brands face budget constraints, they can use an optimizer to identify all campaigns performing below a certain CAC threshold. If they need to cut 25 percent of their media budget, the system can recommend which campaigns to reduce and how to reallocate remaining budget for maximum efficiency.

Conversely, if a brand has an extra half-million dollars to invest, they can use the optimizer to quickly determine how to deploy it across their channel mix to optimize for new customers, CAC targets, or pure top-line revenue growth.

## The Holiday Season Inefficiency Problem

One interesting insight from analyzing thousands of campaigns is that there's massive wasted spend during holiday seasons. Brands often try to chase revenue or ROAS right during peak moments, but this is largely ineffective. There's a relationship between the ratio of top-funnel to bottom-funnel spend, plus a critical time component.

Bottom-funnel campaigns will naturally perform better during seasonal moments regardless of support. But top-funnel campaigns need to begin 45 days in advance to build the right foundation. The optimizer accounts for this timing and saturation dynamics. You can't expect to spend $2 million on top funnel and get a $10 million return just because spending $2 got you $10.

## The Onboarding Process

Getting started with Prescient takes just 12 to 16 minutes. You connect your Shopify or other e-commerce platform, your Google Analytics, and all your ad channels. The platform integrates with everything from Tatari and Mountain for TV, to Neon Pixel and Tenuity, to the Trade Desk and PodScribe, to Meta, Google, TikTok, Reddit, and more. Amazon Seller Central and Vendor Central data integrate as well.

For channels that don't have API connections, like out-of-home or direct mail, you can simply upload data via Google Sheets with basic information like campaign name and spend.

Within a week, all your data is populated and ready to go. Importantly, no humans tinker with your model on your behalf. It's all learned from your historical data, and Prescient doesn't share other brands' data with you. The core belief about how your business generates revenue comes entirely from your own data.

## The Most Mind-Blowing Features

The optimizer is arguably the most impressive feature. When executives sit in board meetings knowing there's a halo effect but struggling to quantify it, the daily recalibration of Prescient solves this problem. The fact that models can run every single day, going down to individual campaign granularity while maintaining powerful machine learning insights, feels almost like MTA but with holistic view.

The ability to run unlimited scenario planning at no additional cost is also revolutionary. Click a button, increase a budget by $500,000, and in 20-40 seconds receive a recommended media plan showing saturation, current spending versus recommended spending, and the predicted impact.

Another recent feature is scenario tracking. Create an optimization scenario, then accept or reject changes with reasoning so the model can learn. The platform then tracks during the actual flight whether campaigns are on track to hit forecasts. Instead of waiting until the end of the month, you get a heads-up in the middle of execution whether things are working as predicted.

## Interesting Channel Findings

AppLovin surprised many observers. On average, brands spending on AppLovin see 5 to 7 percent incremental budget allocation—additional spend on top of existing budgets, not reallocated from elsewhere.

YouTube has proven to be a strong performer despite always being a view-through channel. Cody from Jones Road ran a holdout test, found YouTube was incremental, used Prescient's optimizer to scale it, then validated the results again with another test. All the numbers matched up, proving the textbook play works. With nearly 2,000 YouTube campaigns analyzed, YouTube consistently shows as a reliable channel for brands with strong content and strategy.

Linear TV has been interesting for beauty brands in particular. One beauty brand showed that 47 percent of their revenue over the past year came from media spend driving retail sales. Prescient backtested this with 93.5 percent accuracy over 90 days. These insights empower chief digital officers to sit down in board meetings with well-researched, thoughtful conversations about strategy changes.

## The Future of Attribution

The future likely involves consolidation of different measurement forms—MMM plus incrementality, augmented by AI-driven creative-level analytics and automation. Rather than full automation, you'll see augmentation of human decision-making at key checkpoints.

The vision is conversational and curated—like waking up to a newspaper that tells you what matters. For example, if you're planning to buy a house soon, you'd want to know when the Fed plans to hike rates thirty days out so you can act accordingly. Similarly, AI will surface key insights that matter for your business decisions right now.

The industry is moving toward universal media measurement powered by augmented automation, where portions of buying get automated but human judgment remains central at critical junctures. Clean, simple, conversational reporting replaces dense dashboards.

## At Scale

Prescient currently tracks north of $200 million in media spend monthly across brands in e-commerce, retail, gambling, airlines, hotels, and financial services. The models have diversified to support different verticals, not just traditional DTC.

For anyone curious about MMM evaluation or wanting to understand what questions to ask when assessing different MMM platforms, the invitation is open. Every model is different and the math matters. You can reach Mike True at Mike at prescient AI.com or find Prescient at prescient AI.com.