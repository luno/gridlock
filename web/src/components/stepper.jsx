import React from 'react';
import PropTypes from 'prop-types';
import DOMPurify from 'dompurify';

function sanitizeHtml(html) {
  return html
    ? DOMPurify.sanitize(html, {
        ALLOWED_TAGS: ['span', 'p'],
        ALLOWED_ATTR: ['class'],
      })
    : '';
}

class Stepper extends React.Component {
  constructor (props) {
    super(props);
    this.state = {
    };
  }

  stepChanged (index) {
    this.props.changeCallback(index);
  }

  render () {
    return (
      <ol className="stepper">
        {
          this.props.steps.map((step, index) => {
            let className = this.props.selectedStep === index ? 'is-current' : undefined;
            className = className || this.props.selectedStep > index ? 'is-lower' : undefined;
            if (className === 'is-lower' && this.props.selectedStep > index) { className += ' show-bar'; }
            let stepName = step.name ? step.name.trim() : undefined;
            stepName = stepName || '&nbsp;';
            return (
              <li key={index} className={className} data-step=" " onClick={() => this.stepChanged(index)} dangerouslySetInnerHTML={{ __html: sanitizeHtml(stepName) }}>
              </li>
            );
          })
        }
      </ol>
    );
  }
}

Stepper.propTypes = {
  steps: PropTypes.array.isRequired,
  selectedStep: PropTypes.number.isRequired,
  changeCallback: PropTypes.func.isRequired
};

export default Stepper;
