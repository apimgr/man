// Package graphql provides GraphQL API for casman.
// See AI.md for details.
package graphql

// Schema is the GraphQL schema definition.
const Schema = `
type Query {
  # Get a man page by name
  manPage(name: String!, section: String, platform: String): ManPage

  # Search man pages
  search(
    query: String!
    section: String
    platform: String
    page: Int
    limit: Int
  ): SearchResponse!

  # List all sections
  sections: [Section!]!

  # List all platforms
  platforms: [Platform!]!

  # Compare a man page across platforms
  compare(name: String!, section: String): CompareResult

  # Get database statistics
  stats: Stats!

  # Whatis lookup
  whatis(name: String!): WhatisResult

  # Apropos search
  apropos(query: String!): [WhatisResult!]!
}

type ManPage {
  id: ID!
  name: String!
  section: String!
  title: String!
  platform: String!
  distro: String
  version: String
  language: String

  synopsis: String
  description: String

  contentHTML: String!
  contentText: String!
  contentMarkdown: String!
  contentRaw: String!

  seeAlso: [SeeAlsoEntry!]!
  otherPlatforms: [String!]!

  updatedAt: String!
  createdAt: String!
}

type SeeAlsoEntry {
  name: String!
  section: String!
  url: String!
}

type SearchResponse {
  query: String!
  total: Int!
  page: Int!
  limit: Int!
  results: [SearchResult!]!
  suggestions: [String!]
}

type SearchResult {
  name: String!
  section: String!
  title: String!
  platform: String!
  distro: String
  snippet: String
  score: Float!
  url: String!
}

type Section {
  id: String!
  name: String!
  description: String!
  count: Int!
}

type Platform {
  id: String!
  name: String!
  description: String!
  count: Int!
}

type CompareResult {
  name: String!
  section: String!
  platforms: [ComparePlatform!]!
}

type ComparePlatform {
  platform: String!
  title: String!
  synopsis: String
  options: [String!]
  contentHTML: String
  available: Boolean!
}

type Stats {
  totalPages: Int!
  totalSections: Int!
  totalPlatforms: Int!
  totalLanguages: Int!
  bySection: [SectionCount!]!
  byPlatform: [PlatformCount!]!
  lastUpdated: String!
}

type SectionCount {
  section: String!
  count: Int!
}

type PlatformCount {
  platform: String!
  count: Int!
}

type WhatisResult {
  name: String!
  section: String!
  title: String!
  platform: String!
}
`
