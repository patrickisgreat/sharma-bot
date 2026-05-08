# Limited Supply: Vibe Coding with Billy Howell

I spent the last nine years building, scaling, and investing in brands. Through this show and my weekly newsletter, I'm here to share everything I've learned—the wins, the losses, the experiments, the tactics, and the insights, all so you can unlock your next $100,000 in revenue.

Today we've got a really fun interview with Billy Howell, known as the vibe coder in the e-commerce world. He's a great follow on Twitter if you're interested in vibe coding or building apps to automate things and make processes faster. For example, if you have a back-in-stock app that's charging you $50 a month, or a bundle-building app that charges you as you scale your monthly orders, those are all things that Billy can help you turn into an app that you build once and you no longer pay monthly fees for.

There's a lot of interesting ways that vibe coding, which is really just AI-assisted coding, can make its way into the world of e-commerce. In this episode, I spent about 40 minutes interviewing Billy, talking about what vibe coding is, how to get started, how Billy got started (which illustrates how easy it is for somebody to get going), what e-commerce brands can gain out of it, who in e-commerce brands—whether it's operations, supply chain, marketing, or the creative team—can take advantage of it, and then we dove into the exact software and how you do it.

Billy is known as the vibe coder in our world of e-commerce and is a great follow on Twitter. His company is called Stupid Simple Apps. You can reach him by email or through his various platforms.

**Billy's Background and Path to Vibe Coding**

Billy started out fascinated by apps growing up, watching the iPod Touch and the App Store blow up. He was always interested in building something with code but didn't have a technical background. He took a coding class in college and got a C plus. He worked in DOD consulting in DC during his early career, then quit during the pandemic to start his own business doing SEO consulting with startups.

When ChatGPT came out, he naturally started using it to help his clients and improve his own workflows. He was pasting in Google Apps Scripts, asking ChatGPT to debug them, and then testing them. When Replit came out, it was transformative for him. If ChatGPT was like a zero to one for his coding, Replit was like a one to ten because he was able to contain all those things in self-deployed apps and iterate at the speed of light.

From there, it became an obsession where he was building every idea he'd had over the last twenty years. He threw ideas into Replit, Cursor, and Lovable and saw what he could bring to life. He continued to build useful stuff for his existing clients.

This all came to a point where he was looking for new SEO clients on Upwork and saw a guy who needed something unrelated to SEO—something with Airtable data entry. Looking at his stack, Billy realized this was going to come out to $250 a month in software for something really basic, just a data entry app. He recorded a Loom and pitched him a prototype he made in 30 minutes. He didn't say this was built on Replit or explain what AI coding was because the client didn't care. The client just cared if it worked. When the client said he liked Billy's version better, that was the impetus for his agency, Stupid Simple Apps.

**What is Vibe Coding?**

Vibe coding started as a pejorative term making fun of people, but now has a positive connotation. It's really just a way of being flippant about coding and saying instead of planning out a multi-step thing and putting it in a Jira board like a proper software engineer, you're just going to go off vibes and tell the AI to build this. If it doesn't work, you tell it to fix it and keep iterating.

The interesting thing is that you actually do learn by osmosis doing this, not in a traditional way, but by being able to rapidly iterate. When Cursor first came out, Billy was able to start working on two to three apps every weekend. In the previous way of just Googling things on Stack Overflow, it would take him a month to get even close to finishing a project.

This rapid iterative learning, building an app, then building it in a different platform or starting over and building it better using what you've learned, has given Billy a pretty good understanding of React and Node, which is what most people use for web apps. Going into it, he had no idea what any of the folders meant or how they were intended to work, but he learned from observation.

**APIs and Integrations**

An API is usually a library of endpoints. You can think of the endpoints as different stores you go to. Each one is going to have different data or capabilities. One API endpoint might retrieve user data. Another might take data and turn it into an image or transform it in some way.

When you look at tools like make.com or Zapier, any of those nodes that you see usually means there's an API you could use in your app. If you're ever unsure if something can be done or a certain use case can be done, it's good to go shopping in these hubs for solutions. You could also go to somewhere like Rapid API or Appify and take a look at which ones are available.

Basically, how it works is there's a URL and your app sends it a packet of data, including usually an API key, which is what lets you get your unique data or attach you to a billing account. Most APIs require you to be paying to use them. You send the API who you are and what you're asking for, and then it sends back what's called JSON format. You can make your app read that and use it.

There are tons of APIs out there. For example, basketball reference would be really cool to plug in if you're into the NBA, though you'd need to check if it's public. But there are many available, and APIs are useful because instead of you having to store every single data point on your own server, someone else is doing that for you so you can just query it. Beyond data, APIs can also do things like generate videos, generate images, or actually manipulate things. For example, OpenAI has an API where you send it a prompt and it generates back a result and sends it to you.

The secret sauce of every AI coding platform like Cursor, Replit, and Lovable is that they've taken all the models and found out which ones are best at which tasks. They see a request come in and know which model is best at handling it, then send the request there and prompt it in exactly the right way to get the output they want to turn into code.

**Tech Stack for Vibe Coding**

You basically need ChatGPT and Replit. Sometimes Billy will use V0 or Lovable to make a really pretty front end since they make really pretty UI, but at the end of the day, you just need ChatGPT for planning and maybe troubleshooting, and then Replit or Cursor as an alternative.

Cursor is a local coding IDE, so it's going to be a little more intimidating. Whereas something like Replit or Lovable is all in the browser, and all the code is stored in the cloud. You don't have to worry about installing things like dependencies for specific integrations to work on your laptop.

Billy started with Cursor and then went to Replit. Depending on the size and type of task, he'll flip between those two. Cursor is going to be more custom and allow you to do anything you ask. Whereas anything in the cloud is going to use the stacks that it's trained to be best on. So Cursor is better if you want to be really custom, you're doing something that's atypical or has lots of moving parts, or maybe even if you want to go slower. Replit or Lovable are better if you want to get to a quick prototype or proof of concept, something super easy like a database with username and login. But you can build way more complex apps in there.

**Using ChatGPT for Planning**

Billy uses an app called Super Whisper for voice-to-text. He finds that when you're talking to AI, it's more helpful to be verbose, and you're just limited by how fast you can type. So he goes into ChatGPT and describes the app like he would describe it to you in conversation. If he has unknown things or isn't sure how the app should look or how to do something, he asks ChatGPT to research that.

At the end, he asks for what's called a PRD, or product requirements document, which is something borrowed from real software engineers. He just lists all the features, all the screens or pages you need, and the integrations you're going to use. Then he looks at that and pops it into Replit or Cursor and tells it to get to work.

This makes your starting point a lot better because if you start out incorrectly at the gate, which is where a lot of people get stuck, you'll have an idea that comes out looking nothing like what you pictured in your head. Then you're trying to unwind hundreds of lines of code. So you want to spend two minutes to get closer to a fully fleshed-out idea before you start. It'll save you two hours of time.

**Prompting Strategies**

Billy has figured out some tricks to make prompting better and more accurate. One thing he does is in his system prompt in ChatGPT, he asks it to ask him three clarifying questions every time he sends a prompt. That really helps get the AI thinking and gets him thinking too.

When working with an AI coding agent and adding a new feature or fixing something, he explains the problem and says, "Diagnose the problem and find the solution. Then walk me through it. But don't write any code." If you think about it, when you send a request, ChatGPT has a limited amount of bandwidth. It's going to spend some on thinking about the problem and some on coding. But if you just tell it to walk you through the problem, it's going to use all its bandwidth to find the solution. For you, it's going to poke around more because it's not thinking about saving tokens.

If he knows something is going to be a multi-step thing that builds out and comes back to build more, he tells it not to worry about tokens and that you can do this in multiple steps. He suspects that makes a difference.

Another neat thing is that Windsurf and Cursor have the ability to toggle in the chat input whether you want to chat or edit code. You can just cut it off from coding, which is really useful.

If something breaks after the agent has coded something, don't add more code on top of it. Roll it back. Most AI coding software has the ability to roll back to a checkpoint. Roll it back and then try again. Don't keep putting good code on top of bad code. Being disciplined about that makes a really big difference.

**Design and CSS Modifications**

Design in CSS is really easy. In Replit and most of them, you can click a little pin button and select elements on the page. You can manually edit it in the chat, and it'll show you all the CSS values. You can say delete it, modify it in this way, or do something else. Non-technical people can very easily edit the design if you just add them to a repo.

A good practice is to separate the app from the actual landing page or marketing pages. At the beginning of the prompt, specifically say don't build a landing page—this is just the app. Then you can deploy that to app.yoursubdomain. You can make a landing page either separately in a different AI tool or in Webflow or ClickFunnels or whatever you want to use.

**E-Commerce Use Cases**

There are two main problems Billy has seen get solved with vibe coding. One is pulling all your data into one place. This is a huge need for e-commerce operators. Maybe marketing data is in one place, Shopify data is in another. There are so many dashboards. Operators want to connect to those APIs and pull it into one report. Sometimes they throw AI on top of that to get quick insights on trends about marketing spend and how it relates to inventory, or make suggestions based on inventory they're high on and could discount.

The other is that e-commerce operators often have multiple brands. They may have used Shopify for one brand and WooCommerce for another, and maybe those don't play nice together. Making your own custom data visualization for that is useful. Everyone has their own KPIs that they find most important, so bringing data into one place is valuable.

E-commerce operators are shrewd, so they're realizing they shouldn't be paying $50 a month for an out-of-stock widget on Shopify. They can just build their own waitlisting. Anything with inventory management is ripe for this. For example, tracking shipping containers and how that updates how many pre-orders can go out in Shopify just doesn't connect. Every single brand is unique, and their workflows are unique, which means they all have unique problems. Smart e-commerce operators have been using AI to cut out those SaaS costs that stack up and also unify their data.

The reality is that a lot of these platforms will not be talking to each other for the next two to three years, but they will eventually have to because they'll need to. Meanwhile, you can connect these apps without having to use a body to do that. Right now, you have to have a person who's manually doing all this work.

It's not so much a question of if it can be done or if it will be done in the next two years. The data could roll up through various platforms like Looker Studio or DataBox, but because each business has its own unique problems, even these sprawling platforms can't anticipate your needs. And to do exactly what you want, it gets so complicated with filters that you might as well be coding anyway.

**Scrappy Use Cases with High ROI**

The quickest path to getting outsized returns on vibe coding versus making a D2C app is to go B2B first, especially if you've never done either. The average deal size is higher. It's more relationship-based, one-on-one versus D2C where you might have fifty customers all paying ten dollars, making it hard to separate signal from noise on what to build.

Examples of good B2B use cases include AI receptionists for local businesses, building those custom or just setting those up from existing products. Chatbots trained on companies' data are big opportunities. For instance, there was a guy who manages a restaurant group that does charity events every week. He wanted his menus digitized so people could go on the site and say, "I'm in Fort Lauderdale on Thursday and I want steak. Which one can I go to for this charity event?" These little custom AI bots for people's data are definitely useful. Shopping assistants are big in e-commerce too.

**Internal AI Coder Roles**

Billy hasn't really seen an internal full-time role of AI coders or optimizers yet, but you certainly could make the case for yourself at your job to be that person. It just doesn't take that much time to spin up something that's visually impressive and demonstrates value. Since doing podcast interviews, people have DMed Billy saying they sold AI coding solutions to their boss or CEO for twenty to thirty thousand dollars.

The approach was either building out the whole thing or just building a demo. Some people build out the whole thing, while others build out just a demo, make a Loom, and send it to the CEO, or set a meeting with the CEO and demo it there to see if you can get buy-in. Even if they don't like that particular solution, they know they've got a resource they can come back to when they see something in a couple of weeks that could be automated.

**Future of AI and Custom LLMs**

Most of the products out there use generalized LLMs because they're pretty good. Billy thinks we'll see a gold rush maybe of truly custom LLMs that are really trained on your data or data from your industry, making them really good at understanding things like SKUs for fashion rental companies or pricing strategies for painting contractors. That would be a huge value add. It's something he thinks people aren't focusing on at all right now.

This is kind of where the AI stuff started—the idea was that everyone would have their own AI trained on their subject matter. Somehow we've just blown past that and we're using the generalized ones because they're so good and trained on the whole internet theoretically. But he thinks we're going to move towards customized LLMs trained on your specific data and your industry, really good at doing the specific four things you needed to do.

Someone running a brand could just subscribe to a specialized app and use all of its learnings to make sure theirs is perfect. You could do a video call once a week to just brain dump yourself and the AI would ask you smart questions to train itself. It would come back and try to solve problems. You could train it passively over the course of a year and have a really well-trained bot.

**Downloading Your Brain**

People are doing cool stuff with Notion or Obsidian to take in all their apps. Billy uses Granola lately, which records all his calls and lets him search through them. But as far as downloading your whole brain, you would have to take in all your video content, all your written content, all your emails, and synthesize that somehow.

Some people upload podcast transcripts or newsletters, but the issue is that the models don't understand tonality the way that voice recording in ChatGPT does, which is really interesting. There are definitely ways to upload and search, but it's still not perfect. Some of the best systems use backend SQL queries to find relevant information—if someone's asking about a particular topic, you search the transcript database—because the context windows just show you how complex the human brain is. These AI systems are really smart, but they're not anywhere close to what our brains can do.

**The Real Skill**

The real secret sauce is knowing where to turn and navigate and where to go next, versus just grabbing the answer and then figuring out what to do with it. If an answer doesn't work, the AI won't know what to do next. But if you know what to do next, that's the real value. You can think of AI systems as like a really smart, high school intern.

The notion that AI is going to make people dumber could be argued against pretty heavily, the same way it could be for Wikipedia. However, it might make people more impatient. The speed of development has become so fast that waiting five minutes to spin up an app feels like a miracle compared to a year or two ago when you were copying and pasting stuff one file at a time into ChatGPT.

**Personal Use Cases**

There are interesting personal use cases being built. For example, pulling in data from Google Calendar and email to see when was the last time you were at an event or had a meeting with someone, how many times you emailed them this month, and giving them a weighted score showing if this person is really hot or colder. That's not a way to download your brain per se, but it's a really useful use case for creating a knowledge base of your friends and family.

People are also building apps that help them track information they've collected while traveling—connecting people they met to their social media profiles and contact information in a searchable format. There are so many use cases like that which are just going to be instantly solved.

**Getting Started**

Billy's company is called Stupid Simple Apps. You can find them by Googling stupid hyphen simple apps. On Twitter, he's Billy J Howell and has a link in his profile to book a one-on-one strategy call. He also has a YouTube channel with the same name that covers his strategy for building and selling your first apps. He promises it's really approachable and that you can do it, but the key is that people don't follow through. You have to be the one that does it.

There are huge opportunities on Upwork with people sitting there ready to pay you money to build custom software. His content online is very tactical, and someone can look at it on one screen and execute on another screen. A call with Billy is going to be one of the highest ROI calls of your life. His promise is that everyone can do this—he's not that smart and figured it out. You just have to execute on it.

The real crux of where we are now with how bountiful the internet and software are is that you can build anything. You just need someone to tell you to build this and then build that. It's not going to be a hit right away, but you're going to learn from it and work towards something that works. The seventh app someone builds is going to be amazing.