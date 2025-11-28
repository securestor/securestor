import React, { useState } from 'react';
import { Tag, Copy, Check, Search } from 'lucide-react';

export const MetadataViewer = ({ artifact }) => {
  const [copied, setCopied] = useState(false);
  const [searchTerm, setSearchTerm] = useState('');

  if (!artifact) return null;

  const handleCopyMetadata = () => {
    const metadataText = JSON.stringify(artifact.metadata, null, 2);
    navigator.clipboard.writeText(metadataText);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  const filteredMetadata = Object.entries(artifact.metadata).filter(([key, value]) =>
    key.toLowerCase().includes(searchTerm.toLowerCase()) ||
    String(value).toLowerCase().includes(searchTerm.toLowerCase())
  );

  return (
    <div className="bg-white rounded-lg border border-gray-200 p-6">
      <div className="flex items-center justify-between mb-4">
        <h3 className="text-lg font-semibold text-gray-900 flex items-center">
          <Tag className="w-5 h-5 mr-2 text-blue-600" />
          Metadata & Indexing
        </h3>
        <button
          onClick={handleCopyMetadata}
          className="flex items-center space-x-2 px-3 py-2 text-sm text-gray-700 bg-gray-100 rounded-lg hover:bg-gray-200 transition"
        >
          {copied ? (
            <>
              <Check className="w-4 h-4 text-green-600" />
              <span>Copied!</span>
            </>
          ) : (
            <>
              <Copy className="w-4 h-4" />
              <span>Copy JSON</span>
            </>
          )}
        </button>
      </div>

      {/* Search */}
      <div className="mb-4 relative">
        <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 w-4 h-4 text-gray-400" />
        <input
          type="text"
          placeholder="Search metadata..."
          value={searchTerm}
          onChange={(e) => setSearchTerm(e.target.value)}
          className="w-full pl-10 pr-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent text-sm"
        />
      </div>

      {/* Indexing Status */}
      <div className="mb-6 p-4 bg-blue-50 border border-blue-200 rounded-lg">
        <div className="flex items-center justify-between mb-2">
          <span className="text-sm font-medium text-blue-900">Indexing Status</span>
          {artifact.indexing.indexed && (
            <span className="px-2 py-1 text-xs font-medium bg-green-100 text-green-800 rounded">
              Indexed
            </span>
          )}
        </div>
        <div className="space-y-1 text-sm text-blue-800">
          <div className="flex justify-between">
            <span>Full-text Search:</span>
            <span className="font-medium">
              {artifact.indexing.fullTextSearch ? 'Enabled' : 'Disabled'}
            </span>
          </div>
          <div className="flex justify-between">
            <span>Indexed At:</span>
            <span className="font-medium">
              {new Date(artifact.indexing.indexedAt).toLocaleString()}
            </span>
          </div>
        </div>
      </div>

      {/* Metadata Fields */}
      <div className="space-y-3">
        {filteredMetadata.length > 0 ? (
          filteredMetadata.map(([key, value]) => (
            <div key={key} className="p-3 bg-gray-50 rounded-lg">
              <div className="text-xs font-medium text-gray-500 uppercase mb-1">
                {key.replace(/([A-Z])/g, ' $1').trim()}
              </div>
              <div className="text-sm font-medium text-gray-900 break-all">
                {typeof value === 'object' ? JSON.stringify(value) : String(value)}
              </div>
            </div>
          ))
        ) : (
          <div className="text-center py-8 text-gray-500">
            No metadata fields match your search
          </div>
        )}
      </div>

      {/* Re-index Action */}
      <div className="mt-6 pt-6 border-t border-gray-200">
        <button className="w-full px-4 py-2 text-sm font-medium text-blue-600 bg-blue-50 rounded-lg hover:bg-blue-100 transition">
          Re-index Artifact
        </button>
      </div>
    </div>
  );
};