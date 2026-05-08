Welcome back to Limited Supply. Today's episode is going to be completely different from the usual format. Instead of tactical breakdowns or guest interviews, I'm going to walk you through setting up Claude as an AI bot—specifically, an OpenClaw installation on a Mac mini. I'll explain every single step, what each function does, why it's important, and how you can apply these concepts to e-commerce and growth.

I'm going to show you this through multiple angles. There's video recording me right now, I've got my microphone for audio, I'll be screen recording my Mac, and I'm also recording the Mac mini's display on my phone to capture every step before I even access the account. Whether you're listening to audio only or watching the video, you'll be able to follow along one-to-one with me.

Here's my recommendation: try not to be super efficient with your time on this. The rabbit holes with AI are where the gold is. Typically I optimize for efficiency, but in this case, get comfortable, maybe do this between 7 p.m. and 7 a.m., grab your Mac mini with a screen, keyboard, and mouse connected, get your beverage ready, and let's build this together.

I'm starting by turning on the Mac mini and connecting the keyboard and mouse via lightning cable, then switching to Bluetooth. We're setting this up as a new computer. As it initializes and connects to Wi-Fi, I'm going to work in parallel on other setup tasks.

The first thing to understand is that we're creating a completely separate identity for this bot. I'm naming this one Zane Calder. We need to keep everything separate from your personal accounts—separate iCloud, separate Google Apps account, separate email. The reason is security. If this bot somehow gets compromised or is tricked into downloading something malicious, you don't want it to have access to your personal iMessage, email, or other critical accounts. I'm also using a separate virtual credit card that I can kill instantly if needed.

For the Mac account, I'm creating a first and last name specifically for the bot. I'm checking the box to allow the computer account to be reset with Apple password, which gives us a backup option. For the iCloud setup, I'm creating a new Apple account rather than using an existing one, using the same Gmail that will be the bot's primary identity.

While the Mac finishes its setup, I'm going to get several things prepared in parallel. First, we need Node.js and Homebrew installed—these are absolutely necessary for OpenClaw. We'll also need API keys from Anthropic (which owns Claude), and Brave Search API for search functionality. Beyond that, we're going to set up Google Console access and Telegram for communication.

I'm starting by downloading and installing Node.js first. You just run the installer package and follow the prompts. Then we're going to install Homebrew, which we do through Terminal. I'm going to copy the installation line from the Homebrew website and paste it into Terminal. Terminal is basically a code window that most people don't normally use, but it's how we install and run OpenClaw.

When pasting into Terminal, just leave it alone while it runs. Don't click anything. You'll see it working through various packages. Once it's complete and ready for input again, we can move forward.

While that's running, I'm heading to openclaw.ai to get the OpenClaw documentation and also creating a Claude account for free. This is important because throughout the setup, when I get stuck, I'm going to screenshot my Terminal window and ask Claude what to do next. The key phrase to use is "Please see the screenshot below. I need your help to continue setting up OpenClaw. What should I do next?" And critically, you want to add "Make sure you reference the docs at docs.openclaw.ai." This ensures Claude reads the official documentation when helping you troubleshoot. It's one of the most important parts of this process.

Now, back in Terminal, we're going to run the OpenClaw installer. The command will download and install OpenClaw. You navigate using arrow keys for left and right, space bar to select options, and Enter to submit. This is the automated setup process.

Before we get fully into OpenClaw's onboarding, we need to set up our API keys. First, for Anthropic: go to console.anthropic.com, sign in with Google, and select "individual." On the dashboard, go to settings, then billing, and enable auto-reload. I set mine to reload with thirty dollars when it hits twenty. Then go to API keys, create a key, name it something like "Zane Mac Mini," and save that key.

Next, for Brave Search API: go to search.brave.com, sign up with an email, verify it, and go back to Gmail to log in. Once verified, go to billing, add your payment method, then go to available plans. Under Search, hit "Get Started," check the box, and subscribe. Brave gives you five dollars in credits every month anyway. Set your search spend to something like thirty dollars. Once that's done, go to API keys, add a key, name it, and save it.

Back in Terminal, we're going to input these API keys. The OpenClaw setup asks us which provider we want—Anthropic, OpenAI, Gemini, etc. I'm recommending Anthropic because I've found it works best, though I'm just a sample size of one. Choose Anthropic, paste your API key, and select your model. I'm using Sonnet 3.6 because it's cost-effective while still being very capable.

For communication with our bot, we're going to use Telegram. Open Telegram and search for "BotFather"—it's a verified bot with the robot icon. Message it with "\newbot" and follow the prompts. Give your bot a name (Zane Calder) and then a username (something like zane_7362_bot). BotFather will send you a message with a token at the bottom—this is the critical piece. Copy that token and paste it into Terminal when OpenClaw asks for your Telegram bot token. Do not share this token with anyone.

Next, we're going to set up the search provider. When OpenClaw asks, select Brave and paste your Brave Search API key.

For skills configuration, OpenClaw asks if you want to configure them. Say yes. The main skill you want is GOG, which gives you Google integration. This connects the bot to Gmail, Google Sheets, Google Docs, Google Drive, and Google Calendar. This is where the real power comes in—your bot can read your emails, access your documents, modify your spreadsheets, and understand your schedule.

Once you've configured the basic skills, select "boot" for command logger and session summary. Let it install everything. This creates your CloudBot and brings you to a terminal interface where you can talk directly to it.

Now we need to set up Google Console so the bot can actually access your Google account. Create a new project in Google Cloud Console, call it something like "Zane's Mac Mini," and go through the OAuth setup. You'll configure a consent screen, mark it as internal, and create a desktop OAuth client. Download the JSON credentials file—this is what the bot needs to access your Google account.

Once you've downloaded that file and you're back in the bot's terminal interface, it will ask you to add a Google account. Input the email address you want to use (in my case, zane.nick.co), and it will open a browser to authenticate. Log in, and when prompted, click "Always allow" to give the bot access.

After authenticating, you need to enable specific Google APIs. Search for Gmail, Sheets, and Docs in the Google Cloud Console and enable each one. This allows your bot to access these services on your behalf.

The final piece is connecting your bot to Telegram on your phone. Click the link in the last message from BotFather (it's t.me/your_bot_name), click Start, and you'll get a pairing code. Paste that exact code into Terminal, approve the Telegram access request, and you're done. Now you can communicate with your bot entirely through Telegram on your phone.

This whole setup gives you what I like to call a "second brain" for your business. Your bot has read access to your Gmail, Google Drive, calendar, meeting recordings if you set up API connections to Firefiles or Otter, and your Slack. It understands everything you're seeing and doing. When you communicate with it through Telegram, sharing your thoughts and decisions, it shapes its own thinking around how you operate.

The concept here is powerful: you now have another person—Zane in this case—who you can assign tasks to. I've used CloudBots to build landing pages, publisher websites, advertorial software, and genetic focus group research tools. It feels like playing a video game in terms of how easy and fun it is to collaborate with AI on building things.

You can also create multiple bots for different purposes. Ask your main bot to create a subbots for brand tracking, competitor monitoring, ad flagging, financial tracking, or accounts receivable. You can assign different Claude models to different bots depending on the task. Reserve Opus, the most expensive model, for your most complex tasks. Use Sonnet for most of your work since it handles ninety-eight percent of what you'll need at a fraction of the cost.

If you set this up and run into issues, screenshot your Terminal window, ask Claude with that key phrase about referencing the docs, and it will give you exact next steps.