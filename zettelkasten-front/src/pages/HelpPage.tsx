import React, { useState } from 'react';
import { helpCategories, helpSections } from '../data/helpContent';
import { HelpSection } from '../components/help/HelpSection';
import { HelpCategory } from '../types/help';

export function HelpPage() {
  const [activeCategory, setActiveCategory] = useState<HelpCategory['id']>('quickstart');
  const [searchQuery, setSearchQuery] = useState('');

  const filteredSections = helpSections
    .filter(section => {
      const matchesCategory = section.category === activeCategory;
      const matchesSearch = searchQuery === '' ||
        section.title.toLowerCase().includes(searchQuery.toLowerCase()) ||
        section.content.some(content =>
          typeof content.data === 'string' &&
          content.data.toLowerCase().includes(searchQuery.toLowerCase())
        );
      return matchesCategory && matchesSearch;
    })
    .sort((a, b) => a.order - b.order);

  console.log("Asdasdas")
  return (
    <div className="p-8 max-w-6xl mx-auto">
      <header className="mb-8">
        <h1 className="text-4xl font-bold mb-4">Zettelgarden Help Center</h1>
        <p className="text-lg text-gray-600 mb-6">
          Learn how to make the most of your personal knowledge management system.
        </p>

        <div className="mb-6">
          <input
            type="text"
            placeholder="Search help articles..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="w-full max-w-md px-4 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
          />
        </div>
      </header>

      <nav className="mb-8">
        <div className="flex flex-wrap gap-4">
          {helpCategories.map(category => (
            <button
              key={category.id}
              onClick={() => setActiveCategory(category.id)}
              className={`flex items-center gap-2 px-4 py-2 rounded-lg border transition-colors ${activeCategory === category.id
                  ? 'bg-blue-100 border-blue-300 text-blue-800'
                  : 'bg-white border-gray-300 text-gray-700 hover:bg-gray-50'
                }`}
            >
              <span>{category.icon}</span>
              <span className="font-medium">{category.label}</span>
            </button>
          ))}
        </div>

        <div className="mt-4">
          <p className="text-gray-600">
            {helpCategories.find(cat => cat.id === activeCategory)?.description}
          </p>
        </div>
      </nav>

      <main>
        {filteredSections.length > 0 ? (
          <div className="space-y-8">
            {filteredSections.map(section => (
              <HelpSection key={section.id} section={section} />
            ))}
          </div>
        ) : (
          <div className="text-center py-12">
            <p className="text-gray-500">No articles found matching your search.</p>
          </div>
        )}
      </main>

      <footer className="mt-16 p-6 bg-gray-50 rounded-lg">
        <h2 className="text-xl font-semibold mb-2">Still Need Help?</h2>
        <p className="text-gray-600">
          Can't find what you're looking for? Try exploring the interface directly -
          many elements have helpful tooltips when you hover over them.
        </p>
      </footer>
    </div>
  );
}