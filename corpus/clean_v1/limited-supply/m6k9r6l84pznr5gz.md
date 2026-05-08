# Enterprise Resource Planning for D2C Brands

ERP stands for Enterprise Resource Planning, though the acronym could mean almost anything. Today, most vertical software that addresses different aspects of a business and provides a source of truth is usually called an ERP system. There are generic ERP systems that can be applied to multiple industries, but what distinguishes a true ERP is that it includes financial aspects. Without that financial component, it's usually just a point solution rather than a full ERP system.

Fulfil.io is an ERP system specifically built for e-commerce businesses. Think of it as NetSuite designed specifically for e-commerce. The company works with e-commerce businesses ranging from small operations to massive nine-figure merchants like Hexclad, Mejuri, Caraway, and Cadence. The software helps simplify business operations when complexity reaches a level where Excel spreadsheets no longer work.

## When Businesses Need an ERP

People don't necessarily need an ERP at a specific revenue threshold. Rather, they need one when their business reaches sufficient complexity. A simple business with one platform, one fulfillment center, one person producing goods, and minimal SKUs can operate efficiently with spreadsheets. But complexity arises in several ways: operating your own warehouse with receiving, cycle counting, picking, packing, and shipping; having thousands of SKUs across apparel or other product lines; or managing multiple sales channels, 3PLs in different countries, and various locations.

When Native was doing $100 million in revenue, it didn't need an ERP because the business was intentionally structured to be simple. One manufacturer, one fulfillment center, one channel, and very little SKU count made spreadsheets workable. But once a business scales to multiple channels, multiple product lines, and different transaction types, a single source of truth becomes essential.

The smallest customer on Fulfil's platform is in the four to five million revenue range, though they have the necessary complexity to justify it. The largest customer is close to a billion in GMV, based on publicly shared information. The key is that complexity matters more than revenue.

## What an ERP Dashboard Shows

When opening a Fulfil dashboard for a specific SKU, users can see product availability (how much they can still sell), product on hand, items already spoken for via Shopify or wholesale orders, incoming inventory from manufacturers, and outgoing inventory to customers. It shows how much is being manufactured, sales history breakdown by year, every sales order containing the product, every customer who purchased it, every shipment made, every internal transfer, every customer return, every supplier shipment, and every bill of materials used to build the product.

This becomes particularly powerful with products like deodorant that have multiple ingredients. A user can pull up Native Coconut and Vanilla, look back in time, and determine how much of each ingredient is available on hand to know how many units can be made right now. The system can also flag when inventory is running low and suggest issuing a PO to suppliers for coconut oil, knowing that will be a limiting factor three months from now.

For DTC brands, dynamic bundling is crucial. Instead of pre-making bundles and predicting what customers want, merchants can sell bundles created from existing inventory. When a customer buys a bundle at checkout, Fulfil expands it into individual line items for picking, packing, and shipping at the warehouse. The same ingredient can be pooled across multiple products, so coconut oil used in both Coconut and Vanilla and Cucumber and Mint products decrements from one source rather than being allocated separately to each product.

## Building Everything In-House

Fulfil's competitive moat is twofold. First, there's the complexity of creating the software. The team decided to rebuild QuickBooks, a CRM, ShipStation, inventory management, and everything else in-house rather than integrating with existing solutions. Everything lives inside Fulfil, and the company built a better version of ShipStation as part of their WMS.

From day one, the founders wanted to build a complete ERP and WMS system, but this wasn't possible overnight. It took years to build out each capability: inventory management, order management, and everything else. The team didn't want to scale as just an inventory or order management system. The company has 55 employees total, with only two percent in sales. The vast majority are engineers and support staff. The company doesn't have traditional product managers. Instead, engineers are at most one hop away from customers, and the support team performs dual functions of supporting customers and building the product with engineers.

## Integration and Channels

Fulfil is a direct order management system that brings in orders from all different channels. TikTok Shop is one of many channels integrated. The company integrates with hundreds of 3PLs. One example of an obscure integration is Connective, a landing page platform that converts well and generates tens of thousands of orders with one line item each, accepting Stripe or PayPal payments in sixteen-plus currencies.

When implementing, if a sales channel isn't integrated, one hundred percent of Fulfil is available over a REST API. There's a special channel integration API that merchants use. Customers have built custom sales channels and used this API to integrate with Fulfil for unique business requirements.

## Implementation as the Core Challenge

Implementation is the hardest part of scaling Fulfil. Unlike NetSuite, which typically involves third-party implementation partners, Fulfil does all implementations in-house. This is intentional because the company witnessed many failed implementations when consultants didn't understand the space, business, or product. Fulfil has ultra-low failure rates because implementation experts truly understand D2C operations.

Currently, Fulfil has three implementation pods, each including a consultant, implementation specialist, and analyst. Each pod runs at most one to two implementations at a time, so the company is onboarding a maximum of six customers monthly. This is the biggest limiting factor to growth. Even hiring people from supply chain backgrounds or those who've implemented NetSuite or other WMS systems takes months to become domain experts in D2C, understanding how warehouses operate, how they interface with 3PLs, EDI systems, and how merchants sell across different platforms.

New pods take six to twelve months to build. With current growth, two more pods are coming, bringing the total to five pods within the next year. The challenge is that implementations involve on-site work, flying to customer locations, spending weeks at warehouses, launching with customers, and handling all the logistical requirements. Software is easy to scale, but this human-intensive work is purely a function of time.

## Handling Complexity at Scale

When a customer like Walmart wants to integrate, there's crucial domain knowledge involved. If a merchant is shipping from Amazon FBA and sends an FBA tracking number to Walmart, Walmart cancels the account because they don't want Amazon delivering Walmart packages. Fulfil's team can advise merchants to use blank boxes and force Amazon to ship with UPS or FedEx instead. This level of expertise takes time to build and can't be rushed.

The company could have scaled faster early on if it had marketed NetSuite-style inventory or order management systems. But that would have locked them into those perceptions. Customers would expect them to be an inventory system or order management system, not a true ERP. The team patiently built out each capability over years to avoid this trap.

## Failure and Churn

ERP implementations across the board have failure rates between sixty and seventy percent, mostly because they're implemented by people who don't understand the space or business. Fulfil's failure rate is ultra-low due to hands-on implementation expertise.

On customer churn, Fulfil has negative revenue churn overall. One customer had regrettable churn, and two other businesses encountered trouble and closed. There was one unsuccessful implementation last year where Fulfil couldn't get the customer fully onboarded across all their channels. Training and mismatch issues prevented making that customer successful.

## Why Venture Capital Didn't Fit

Fulfil said no to venture capital during peak market conditions when multiple VCs offered term sheets. The founders believed you can't build an ERP system with a venture outcome mindset. The business raised $550,000 from two angel investors in 2017 and bought back all other investor shares to have complete control over the company's direction.

The founders are comfortable running the business long-term because they can prioritize customers over everything else and enjoy solving challenging problems. A business that raises $100 million has to raise the next round at an even larger multiple to justify that capital. DTC is a graveyard of startups that raised money only to be shut down or have prices increased overnight when acquired by private equity or other buyers.

Many customers come to Fulfil after being burned by software that shut down or increased prices suddenly. Products like Stitch and Trade Gecko were good but didn't achieve venture-scale returns and had to be shut down. The landscape for DTC software is littered with failures that could have been prevented with a bootstrap approach.

## Marketing Challenges

The founders acknowledge that marketing is their weakness. All three co-founders come from product backgrounds, so companies are built in the image of their founders. What they're good at becomes the company's strength, and what they're not good at becomes a problem area they need to hire for. Over the last year, they brought on one marketing person who's essentially a one-person army.

Most customers discover Fulfil through word of mouth within the tight DTC community. When people call, they typically either know they need an ERP after failed implementations with NetSuite or other systems, or they're coming with specific problems around complexity, inventory, and bill of materials. They're problem-focused rather than solution-focused initially.

When prospective customers visit the Fulfil website, they struggle to understand what the company does because the marketing message isn't clear. Different buyer types need different information: DTC brands with complex bill of materials, DTC brands with their own 3PLs, DTC brands with omnichannel presence. Videos and screenshots showing specific use cases would help. The website needs improvement, but it's something the team is working on.

## Long-Term Vision Without Venture Returns

The company makes it clear to employees that they should assume their stock options have zero value and an infinitesimally small chance of becoming something. People should join because they think the pay is good and they enjoy the work. If they're joining for stock options to do something magical, they should join another company. This attracts people who are jaded by venture-backed companies.

## Pricing Philosophy

Fulfil never raises prices on existing customers. There's a customer who still pays $1,200 per month after seven years, and no one gets that price again. The company won't raise prices on customers who've been great partners and allowed the company to test features and build products together. Price increases only happen when customers' businesses grow and they buy more from Fulfil: launching EDI with more partners, onboarding additional 3PLs, or purchasing more user seats.

The company uses two pricing models: GMB-based pricing for rapidly scaling businesses that don't want to renegotiate constantly and user-plus-modules pricing for steady, family-run businesses with modest growth. If a business goes from ten million to twenty million in revenue without onboarding new EDI partners or adding seats, they pay the same price.

This contrasts sharply with what happened when ShipStation was acquired by private equity. Prices increased rapidly, and the company changed to API call-based pricing, which created confusion and frustration among customers. Three weeks ago, the Fulfil team spoke with a merchant who'd experienced this with another product acquired by Sage. The vendor sent an invoice doubling the price but offered a fifty percent discount to make it seem unchanged. Within a year, the discount disappeared and prices jumped again, with customers given only a week to decide.

## Why DTC Ecosystem Favors Word of Mouth

The DTC ecosystem is close-knit, with business owners happy to share information with each other. This has helped Fulfil grow through recommendations, but the company acknowledges this can't sustain growth forever. When scaling from three implementation pods to fifteen, word of mouth becomes insufficient. Better marketing and clearer messaging will be necessary to reach entrepreneurs who aren't yet part of the DTC community but have the complexity that Fulfil solves.