## API Design:

For a program, a well-designed API is very important, we design the APIs with principles in `RESTFUL` as follows:

- Keep each API's responsibilities simple.
- Provide APIs that cover the full process life cycle. 
- Use resources as the heart of the API, but they do not have to correspond to an actual data object one-by-one. 
- URLs should include nouns, not verbs. In additional, use plural nouns instead of singular nouns. `E.g.` `/nodes/{name}: GET`.
- Use HTTP response status codes to represent the outcome of operations on resources. 
- Avoid deep URLs paths (Max 4 levels).
- Well-documented, we organize the API with swagger and keep the API synchronized with the documentation through swagger generation tools.
