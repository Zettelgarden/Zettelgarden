import React from 'react';
import { GithubIcon } from '../assets/icons/GithubIcon';
import { TwitterIcon } from '../assets/icons/TwitterIcon';
import { YoutubeIcon } from '../assets/icons/YoutubeIcon';

export const SocialLinks: React.FC<{ className?: string }> = ({
  className = '',
}) => {
  return (
    <div className={`flex items-center space-x-4 ${className}`}>
      <a
        href="https://github.com/NickSavage/Zettelgarden"
        className="text-gray-600 hover:text-gray-900 transition-colors duration-200"
      >
        <GithubIcon />
      </a>
      <a
        href="https://twitter.com/zettelgarden"
        className="text-gray-600 hover:text-gray-900 transition-colors duration-200"
      >
        <TwitterIcon />
      </a>
      <a
        href="https://www.youtube.com/@zettelgarden"
        className="text-gray-600 hover:text-gray-900 transition-colors duration-200"
      >
        <YoutubeIcon />
      </a>
    </div>
  );
};
